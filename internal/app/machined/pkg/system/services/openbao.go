// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"fmt"
	"net/http"
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
)

const (
	// OpenBaoServiceID is the service ID exposed by machined.
	OpenBaoServiceID = "openbao"

	openBaoBinaryPath = "/usr/bin/openbao"
	openBaoConfigPath = "/var/lib/chubo/config/openbao.hcl"
	openBaoDataDir    = "/var/lib/chubo/openbao/data"
	openBaoCertsDir   = "/var/lib/chubo/certs"
)

var _ system.HealthcheckedService = (*OpenBao)(nil)

// OpenBao implements an OS-managed OpenBao service.
type OpenBao struct{}

// ID implements the Service interface.
func (s *OpenBao) ID(runtime.Runtime) string {
	return OpenBaoServiceID
}

// PreFunc implements the Service interface.
func (s *OpenBao) PreFunc(_ context.Context, _ runtime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(openBaoConfigPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(openBaoDataDir, 0o700); err != nil {
		return err
	}

	if err := os.MkdirAll(openBaoCertsDir, 0o700); err != nil {
		return err
	}

	if _, err := os.Stat(openBaoBinaryPath); err != nil {
		return fmt.Errorf("missing openbao binary %q: %w", openBaoBinaryPath, err)
	}

	return nil
}

// PostFunc implements the Service interface.
func (s *OpenBao) PostFunc(runtime.Runtime, events.ServiceState) error {
	return nil
}

// Condition implements the Service interface.
func (s *OpenBao) Condition(runtime.Runtime) conditions.Condition {
	return conditions.WaitForFileToExist(openBaoConfigPath)
}

// DependsOn implements the Service interface.
func (s *OpenBao) DependsOn(runtime.Runtime) []string {
	return []string{machinedServiceID}
}

// Volumes implements the Service interface.
func (s *OpenBao) Volumes(runtime.Runtime) []string {
	return nil
}

// Runner implements the Service interface.
func (s *OpenBao) Runner(r runtime.Runtime) (runner.Runner, error) {
	debug := false

	if r.Config() != nil {
		debug = r.Config().Debug()
	}

	args := &runner.Args{
		ID: s.ID(r),
		ProcessArgs: []string{
			openBaoBinaryPath,
			"server",
			"-config=" + openBaoConfigPath,
		},
	}

	return restart.New(process.NewRunner(
		debug,
		args,
		runner.WithLoggingManager(r.Logging()),
		runner.WithEnv(environment.Get(r.Config())),
		runner.WithOOMScoreAdj(-700),
		runner.WithCgroupPath(constants.CgroupSystem+"/openbao"),
		runner.WithDroppedCapabilities(constants.DefaultDroppedCapabilities),
	),
		restart.WithType(restart.Forever),
	), nil
}

// HealthFunc implements the HealthcheckedService interface.
func (s *OpenBao) HealthFunc(runtime.Runtime) health.Check {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8200/v1/sys/health?standbyok=true&perfstandbyok=true", nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck

		switch resp.StatusCode {
		case http.StatusOK, http.StatusTooManyRequests, 472, 473, http.StatusNotImplemented, http.StatusServiceUnavailable:
			return nil
		default:
			return fmt.Errorf("unexpected openbao health status: %s", resp.Status)
		}
	}
}

// HealthSettings implements the HealthcheckedService interface.
func (s *OpenBao) HealthSettings(runtime.Runtime) *health.Settings {
	settings := health.DefaultSettings
	settings.InitialDelay = 3 * time.Second

	return &settings
}

// APIRestartAllowed implements APIRestartableService.
func (s *OpenBao) APIRestartAllowed(runtime.Runtime) bool {
	return true
}

// APIStartAllowed implements APIStartableService.
func (s *OpenBao) APIStartAllowed(runtime.Runtime) bool {
	return true
}

// APIStopAllowed implements APIStoppableService.
func (s *OpenBao) APIStopAllowed(runtime.Runtime) bool {
	return true
}
