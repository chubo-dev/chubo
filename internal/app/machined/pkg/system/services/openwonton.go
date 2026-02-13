// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"os"
	"path/filepath"
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
	// OpenWontonServiceID is the service ID exposed by machined.
	OpenWontonServiceID = "openwonton"

	openWontonBinaryPath = "/var/lib/chubo/bin/openwonton"
	openWontonConfigPath = "/var/lib/chubo/config/openwonton.hcl"
	openWontonDataDir    = "/var/lib/chubo/openwonton"
	openWontonGlibcDir   = "/var/lib/chubo/lib/openwonton"
	openWontonFallback   = "/usr/bin/init"
)

var _ system.HealthcheckedService = (*OpenWonton)(nil)

// OpenWonton implements an OS-managed Nomad-compatible service.
type OpenWonton struct{}

// ID implements the Service interface.
func (s *OpenWonton) ID(runtime.Runtime) string {
	return OpenWontonServiceID
}

// PreFunc implements the Service interface.
func (s *OpenWonton) PreFunc(ctx context.Context, r runtime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(openWontonBinaryPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(openWontonConfigPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(openWontonDataDir, 0o700); err != nil {
		return err
	}

	if err := os.MkdirAll(openWontonGlibcDir, 0o755); err != nil {
		return err
	}

	if err := EnsureChuboServiceTLSMaterial(ctx, r, OpenWontonServiceID); err != nil {
		return err
	}

	// Prefer real openwonton release artifacts (wonton + glibc bundle from the OCI image tar).
	// Fall back to the local mock binary path if artifact install is unavailable (for example
	// network-restricted boots).
	if err := ensureOpenWontonRuntime(ctx, openWontonBinaryPath, openWontonGlibcDir); err == nil {
		return nil
	}

	return ensureServiceBinary(openWontonBinaryPath, openWontonFallback)
}

// PostFunc implements the Service interface.
func (s *OpenWonton) PostFunc(runtime.Runtime, events.ServiceState) error {
	return nil
}

// Condition implements the Service interface.
func (s *OpenWonton) Condition(r runtime.Runtime) conditions.Condition {
	return conditions.WaitForAll(
		timeresource.NewSyncCondition(r.State().V1Alpha2().Resources()),
		network.NewReadyCondition(r.State().V1Alpha2().Resources(), network.AddressReady, network.HostnameReady),
		conditions.WaitForFileToExist(openWontonConfigPath),
	)
}

// DependsOn implements the Service interface.
func (s *OpenWonton) DependsOn(runtime.Runtime) []string {
	return []string{machinedServiceID}
}

// Volumes implements the Service interface.
func (s *OpenWonton) Volumes(runtime.Runtime) []string {
	return nil
}

// Runner implements the Service interface.
func (s *OpenWonton) Runner(r runtime.Runtime) (runner.Runner, error) {
	debug := false

	if r.Config() != nil {
		debug = r.Config().Debug()
	}

	loader, libraryPath, err := openWontonLoaderAndLibraryPath(openWontonGlibcDir)
	if err != nil {
		return nil, err
	}

	args := &runner.Args{
		ID: s.ID(r),
		ProcessArgs: []string{
			loader,
			"--library-path",
			libraryPath,
			openWontonBinaryPath,
			"agent",
			"-config",
			openWontonConfigPath,
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
		runner.WithCgroupPath(constants.CgroupSystem+"/openwonton"),
		runner.WithDroppedCapabilities(constants.DefaultDroppedCapabilities),
	),
		restart.WithType(restart.Forever),
	), nil
}

// HealthFunc implements the HealthcheckedService interface.
func (s *OpenWonton) HealthFunc(runtime.Runtime) health.Check {
	return func(ctx context.Context) error {
		client, err := NewChuboServiceHTTPClient(OpenWontonServiceID, 2*time.Second)
		if err != nil {
			return err
		}

		return simpleHealthCheckWithClient(ctx, "https://127.0.0.1:4646/v1/status/leader", client)
	}
}

// HealthSettings implements the HealthcheckedService interface.
func (s *OpenWonton) HealthSettings(runtime.Runtime) *health.Settings {
	settings := health.DefaultSettings
	settings.InitialDelay = 3 * time.Second

	return &settings
}

// APIRestartAllowed implements APIRestartableService.
func (s *OpenWonton) APIRestartAllowed(runtime.Runtime) bool {
	return true
}

// APIStartAllowed implements APIStartableService.
func (s *OpenWonton) APIStartAllowed(runtime.Runtime) bool {
	return true
}

// APIStopAllowed implements APIStoppableService.
func (s *OpenWonton) APIStopAllowed(runtime.Runtime) bool {
	return true
}
