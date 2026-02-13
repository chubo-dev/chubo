// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	// OpenGyozaServiceID is the service ID exposed by machined.
	OpenGyozaServiceID = "opengyoza"

	openGyozaBinaryPath = "/var/lib/chubo/bin/opengyoza"
	openGyozaConfigPath = "/var/lib/chubo/config/opengyoza.hcl"
	openGyozaRuntimeCfg = "/var/lib/chubo/config/opengyoza.runtime.hcl"
	openGyozaDataDir    = "/var/lib/chubo/opengyoza"
	openGyozaFallback   = "/usr/bin/init"
)

var openGyozaRelease = serviceReleaseBinary{
	ServiceName: "opengyoza",
	// opengyoza publishes Consul-compatible release artifacts as the `gyoza` binary.
	Version:  "v1.6.4",
	ZipEntry: "gyoza",
	AssetURLs: map[string]string{
		"amd64": "https://github.com/opengyoza/opengyoza/releases/download/v1.6.4/gyoza_1.6.4_linux_amd64.zip",
		"arm64": "https://github.com/opengyoza/opengyoza/releases/download/v1.6.4/gyoza_1.6.4_linux_arm64.zip",
	},
}

var _ system.HealthcheckedService = (*OpenGyoza)(nil)

// OpenGyoza implements an OS-managed Consul-compatible service.
type OpenGyoza struct{}

// ID implements the Service interface.
func (s *OpenGyoza) ID(runtime.Runtime) string {
	return OpenGyozaServiceID
}

// PreFunc implements the Service interface.
func (s *OpenGyoza) PreFunc(ctx context.Context, r runtime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(openGyozaBinaryPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(openGyozaConfigPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(openGyozaDataDir, 0o700); err != nil {
		return err
	}

	if err := EnsureChuboServiceTLSMaterial(ctx, r, OpenGyozaServiceID); err != nil {
		return err
	}

	// Prefer real opengyoza release artifacts. Fallback to the local mock binary path
	// if artifact install is unavailable (for example network-restricted boots).
	if err := ensureServiceBinaryWithRelease(ctx, openGyozaBinaryPath, openGyozaFallback, openGyozaRelease); err != nil {
		return err
	}

	return ensureOpenGyozaRuntimeConfig(ctx)
}

// PostFunc implements the Service interface.
func (s *OpenGyoza) PostFunc(runtime.Runtime, events.ServiceState) error {
	return nil
}

// Condition implements the Service interface.
func (s *OpenGyoza) Condition(r runtime.Runtime) conditions.Condition {
	return conditions.WaitForAll(
		timeresource.NewSyncCondition(r.State().V1Alpha2().Resources()),
		network.NewReadyCondition(r.State().V1Alpha2().Resources(), network.AddressReady, network.HostnameReady),
		conditions.WaitForFileToExist(openGyozaConfigPath),
	)
}

// DependsOn implements the Service interface.
func (s *OpenGyoza) DependsOn(runtime.Runtime) []string {
	return []string{machinedServiceID}
}

// Volumes implements the Service interface.
func (s *OpenGyoza) Volumes(runtime.Runtime) []string {
	return nil
}

// Runner implements the Service interface.
func (s *OpenGyoza) Runner(r runtime.Runtime) (runner.Runner, error) {
	debug := false

	if r.Config() != nil {
		debug = r.Config().Debug()
	}

	args := &runner.Args{
		ID: s.ID(r),
		ProcessArgs: []string{
			openGyozaBinaryPath,
			"agent",
			"-config-file",
			openGyozaRuntimeCfg,
		},
	}

	return restart.New(process.NewRunner(
		debug,
		args,
		runner.WithLoggingManager(r.Logging()),
		runner.WithEnv(append(
			environment.Get(r.Config()),
			constants.EnvXDGRuntimeDir,
		)),
		runner.WithOOMScoreAdj(-700),
		runner.WithCgroupPath(constants.CgroupSystem+"/opengyoza"),
		runner.WithDroppedCapabilities(constants.DefaultDroppedCapabilities),
	),
		restart.WithType(restart.Forever),
	), nil
}

// HealthFunc implements the HealthcheckedService interface.
func (s *OpenGyoza) HealthFunc(runtime.Runtime) health.Check {
	return func(ctx context.Context) error {
		client, err := NewChuboServiceHTTPClient(OpenGyozaServiceID, 2*time.Second)
		if err != nil {
			return err
		}

		return simpleHealthCheckWithClient(ctx, "https://127.0.0.1:8500/v1/status/leader", client)
	}
}

// HealthSettings implements the HealthcheckedService interface.
func (s *OpenGyoza) HealthSettings(runtime.Runtime) *health.Settings {
	settings := health.DefaultSettings
	settings.InitialDelay = 3 * time.Second

	return &settings
}

// APIRestartAllowed implements APIRestartableService.
func (s *OpenGyoza) APIRestartAllowed(runtime.Runtime) bool {
	return true
}

// APIStartAllowed implements APIStartableService.
func (s *OpenGyoza) APIStartAllowed(runtime.Runtime) bool {
	return true
}

// APIStopAllowed implements APIStoppableService.
func (s *OpenGyoza) APIStopAllowed(runtime.Runtime) bool {
	return true
}

func ensureOpenGyozaRuntimeConfig(ctx context.Context) error {
	base, err := os.ReadFile(openGyozaConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read opengyoza base config %q: %w", openGyozaConfigPath, err)
	}

	ip, err := defaultOutboundIPv4(ctx)
	if err != nil {
		return err
	}

	lines := strings.Split(string(base), "\n")
	out := make([]string, 0, len(lines)+2)

	out = append(out, fmt.Sprintf("bind_addr = %q", ip))
	out = append(out, fmt.Sprintf("advertise_addr = %q", ip))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "bind_addr") || strings.HasPrefix(trimmed, "advertise_addr") {
			continue
		}

		out = append(out, line)
	}

	content := strings.TrimSpace(strings.Join(out, "\n")) + "\n"

	partial := openGyozaRuntimeCfg + ".part"
	if err := os.WriteFile(partial, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write opengyoza runtime config %q: %w", partial, err)
	}

	if err := os.Rename(partial, openGyozaRuntimeCfg); err != nil {
		_ = os.Remove(partial)

		return fmt.Errorf("failed to place opengyoza runtime config %q: %w", openGyozaRuntimeCfg, err)
	}

	return nil
}

func defaultOutboundIPv4(ctx context.Context) (string, error) {
	// Consul refuses to start if it sees multiple private IPs without an explicit bind/advertise
	// address. Pick the address used for the default route (same trick as many agents).
	const target = "1.1.1.1:80"

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		// Keep this bounded, but allow the network stack a moment right after boot.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	for {
		conn, err := (&net.Dialer{}).DialContext(ctx, "udp4", target)
		if err == nil {
			udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
			_ = conn.Close()
			if ok && udpAddr.IP != nil && !udpAddr.IP.IsUnspecified() {
				return udpAddr.IP.String(), nil
			}

			return "", fmt.Errorf("failed to derive default outbound IP (local=%v)", udpAddr)
		}

		// Retry until timeout/deadline.
		if time.Now().After(deadline) {
			return "", fmt.Errorf("failed to derive default outbound IP: %w", err)
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("failed to derive default outbound IP: %w", ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
}
