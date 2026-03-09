// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/events"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/health"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/runner"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/runner/process"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/runner/restart"
	"github.com/chubo-dev/chubo/internal/pkg/environment"
	"github.com/chubo-dev/chubo/pkg/conditions"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	timeresource "github.com/chubo-dev/chubo/pkg/machinery/resources/time"
)

const (
	// DockerServiceID is the service ID exposed by machined.
	DockerServiceID = "docker"

	dockerBinaryPath      = "/var/lib/chubo/bin/dockerd"
	dockerArtifactURLPath = "/var/lib/chubo/config/docker.artifact_url"
	dockerDataDir         = "/var/lib/chubo/docker"
	dockerExecRoot        = "/var/run/docker"
	dockerSocketPath      = "/var/run/docker.sock"
	dockerPidPath         = "/var/run/docker.pid"
)

var dockerRelease = serviceReleaseTarGzBinary{
	ServiceName: "docker",
	Version:     "v28.1.1",
	TarEntry:    "docker/dockerd",
	AssetURLs: map[string]string{
		"amd64": "https://download.docker.com/linux/static/stable/x86_64/docker-28.1.1.tgz",
		"arm64": "https://download.docker.com/linux/static/stable/aarch64/docker-28.1.1.tgz",
	},
}

var _ system.HealthcheckedService = (*Docker)(nil)

// Docker implements an OS-managed Docker daemon for OpenWonton/Nomad docker workloads.
type Docker struct{}

// ID implements the Service interface.
func (d *Docker) ID(runtime.Runtime) string {
	return DockerServiceID
}

// PreFunc implements the Service interface.
func (d *Docker) PreFunc(ctx context.Context, r runtime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(dockerBinaryPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dockerArtifactURLPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(dockerDataDir, 0o700); err != nil {
		return err
	}

	if err := os.MkdirAll(dockerExecRoot, 0o700); err != nil {
		return err
	}

	release := dockerRelease

	// Allow dev/test harnesses to override where we fetch release assets (e.g. local mirrors for airgapped CI).
	if raw, err := os.ReadFile(dockerArtifactURLPath); err == nil {
		override := strings.TrimSpace(string(raw))
		if override != "" {
			release = dockerRelease
			release.AssetURLs = make(map[string]string, len(dockerRelease.AssetURLs))
			for k, v := range dockerRelease.AssetURLs {
				release.AssetURLs[k] = v
			}

			release.AssetURLs[goruntime.GOARCH] = override
		}
	}

	// Keep an existing installed dockerd usable if release download/cache is temporarily unavailable.
	if err := ensureServiceBinaryWithTarGzRelease(ctx, dockerBinaryPath, dockerBinaryPath, release); err != nil {
		return err
	}

	_ = os.Remove(dockerSocketPath)
	_ = os.Remove(dockerPidPath)

	return nil
}

// PostFunc implements the Service interface.
func (d *Docker) PostFunc(runtime.Runtime, events.ServiceState) error {
	return nil
}

// Condition implements the Service interface.
func (d *Docker) Condition(r runtime.Runtime) conditions.Condition {
	// Docker release downloads require working network + time sync for HTTPS validation.
	return conditions.WaitForAll(
		timeresource.NewSyncCondition(r.State().V1Alpha2().Resources()),
		network.NewReadyCondition(r.State().V1Alpha2().Resources(), network.AddressReady, network.HostnameReady),
	)
}

// DependsOn implements the Service interface.
func (d *Docker) DependsOn(runtime.Runtime) []string {
	return []string{"containerd"}
}

// Volumes implements the Service interface.
func (d *Docker) Volumes(runtime.Runtime) []string {
	return nil
}

// Runner implements the Service interface.
func (d *Docker) Runner(r runtime.Runtime) (runner.Runner, error) {
	debug := false

	if r.Config() != nil {
		debug = r.Config().Debug()
	}

	args := &runner.Args{
		ID: d.ID(r),
		ProcessArgs: []string{
			dockerBinaryPath,
			"--host=unix://" + dockerSocketPath,
			"--containerd=" + constants.SystemContainerdAddress,
			"--data-root=" + dockerDataDir,
			"--exec-root=" + dockerExecRoot,
			"--pidfile=" + dockerPidPath,
			"--userland-proxy=false",
		},
	}

	envVars := setEnvVar(environment.Get(r.Config()), "PATH", filepath.Dir(dockerBinaryPath)+":"+constants.PATH)
	envVars = append(envVars, constants.EnvXDGRuntimeDir)

	return restart.New(process.NewRunner(
		debug,
		args,
		runner.WithLoggingManager(r.Logging()),
		runner.WithEnv(envVars),
		runner.WithOOMScoreAdj(-700),
		runner.WithCgroupPath(constants.CgroupSystem+"/docker"),
		runner.WithDroppedCapabilities(constants.DefaultDroppedCapabilities),
	),
		restart.WithType(restart.Forever),
	), nil
}

// HealthFunc implements the HealthcheckedService interface.
func (d *Docker) HealthFunc(runtime.Runtime) health.Check {
	return func(ctx context.Context) error {
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer

				return dialer.DialContext(ctx, "unix", dockerSocketPath)
			},
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   2 * time.Second,
		}

		if err := simpleHealthCheckWithClient(ctx, "http://docker/_ping", client); err != nil {
			return fmt.Errorf("docker daemon health check failed: %w", err)
		}

		return nil
	}
}

// HealthSettings implements the HealthcheckedService interface.
func (d *Docker) HealthSettings(runtime.Runtime) *health.Settings {
	settings := health.DefaultSettings
	settings.InitialDelay = 2 * time.Second
	settings.Timeout = 2 * time.Second

	return &settings
}
