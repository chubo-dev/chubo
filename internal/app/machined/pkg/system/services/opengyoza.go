// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/events"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/health"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/runner"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/runner/process"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/runner/restart"
	"github.com/siderolabs/talos/internal/pkg/environment"
	"github.com/siderolabs/talos/pkg/conditions"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
	timeresource "github.com/siderolabs/talos/pkg/machinery/resources/time"
)

const (
	// OpenGyozaServiceID is the service ID exposed by machined.
	OpenGyozaServiceID = "opengyoza"

	openGyozaBinaryPath = "/var/lib/chubo/bin/opengyoza"
	openGyozaConfigPath = "/var/lib/chubo/config/opengyoza.hcl"
	openGyozaDataDir    = "/var/lib/chubo/opengyoza"
	openGyozaFallback   = "/usr/bin/init"
)

var _ system.HealthcheckedService = (*OpenGyoza)(nil)

// OpenGyoza implements an OS-managed Consul-compatible service.
type OpenGyoza struct{}

// ID implements the Service interface.
func (s *OpenGyoza) ID(runtime.Runtime) string {
	return OpenGyozaServiceID
}

// PreFunc implements the Service interface.
func (s *OpenGyoza) PreFunc(context.Context, runtime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(openGyozaBinaryPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(openGyozaConfigPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(openGyozaDataDir, 0o700); err != nil {
		return err
	}

	// Until real artifacts are installed, boot with a fallback executable (machined binary)
	// which dispatches to the opengyoza mock mode by argv0.
	return ensureServiceBinary(openGyozaBinaryPath, openGyozaFallback)
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
			openGyozaConfigPath,
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
		return simpleHealthCheck(ctx, "http://127.0.0.1:8500/v1/status/leader")
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
