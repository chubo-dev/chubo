// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/siderolabs/talos/internal/app/machined/pkg/system"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system/services"
	chubores "github.com/siderolabs/talos/pkg/machinery/resources/chubo"
	"github.com/siderolabs/talos/pkg/machinery/resources/config"
	v1alpha1res "github.com/siderolabs/talos/pkg/machinery/resources/v1alpha1"
)

const (
	openGyozaConfigPath = "/var/lib/chubo/config/opengyoza.hcl"
	openGyozaRolePath   = "/var/lib/chubo/config/opengyoza.role"
	openGyozaRoleServer = "server"
)

// OpenGyozaServiceManager is the interface to v1alpha1 service manager.
type OpenGyozaServiceManager interface {
	IsRunning(id string) (system.Service, bool, error)
	Load(services ...system.Service) []string
	Start(serviceIDs ...string) error
	Stop(ctx context.Context, serviceIDs ...string) error
}

// OpenGyozaServiceController starts/stops the OS-managed opengyoza service from machine config intent.
type OpenGyozaServiceController struct {
	V1Alpha1ServiceManager OpenGyozaServiceManager
}

// Name implements controller.Controller interface.
func (ctrl *OpenGyozaServiceController) Name() string {
	return "chubo.OpenGyozaServiceController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenGyozaServiceController) Inputs() []controller.Input {
	return []controller.Input{
		{
			Namespace: config.NamespaceName,
			Type:      config.MachineConfigType,
			ID:        optional.Some(config.ActiveID),
			Kind:      controller.InputWeak,
		},
		{
			Namespace: v1alpha1res.NamespaceName,
			Type:      v1alpha1res.ServiceType,
			ID:        optional.Some(services.OpenGyozaServiceID),
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *OpenGyozaServiceController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenGyozaStatusType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenGyozaServiceController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		r.StartTrackingOutputs()

		mc, err := safe.ReaderGetByID[*config.MachineConfig](ctx, r, config.ActiveID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting machine config: %w", err)
		}

		configured, role, err := openGyozaConfigured(mc)
		if err != nil {
			return fmt.Errorf("error reading opengyoza intent: %w", err)
		}

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenGyozaServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting opengyoza service state: %w", err)
		}

		if configured {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenGyozaServiceID)
			if runErr != nil {
				ctrl.V1Alpha1ServiceManager.Load(&services.OpenGyoza{})
				running = false
			}

			if !running {
				if err := ctrl.V1Alpha1ServiceManager.Start(services.OpenGyozaServiceID); err != nil {
					return fmt.Errorf("error starting opengyoza service: %w", err)
				}
			}
		} else {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenGyozaServiceID)
			if runErr == nil && running {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.OpenGyozaServiceID); err != nil {
					return fmt.Errorf("error stopping opengyoza service: %w", err)
				}
			}
		}

		running := false
		healthy := false

		if svcRes != nil {
			running = svcRes.TypedSpec().Running
			healthy = svcRes.TypedSpec().Healthy
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenGyozaStatus(), func(res *chubores.OpenGyozaStatus) error {
			res.TypedSpec().Configured = configured
			res.TypedSpec().Role = role
			res.TypedSpec().Running = running
			res.TypedSpec().Healthy = healthy

			return nil
		}); err != nil {
			return fmt.Errorf("error updating opengyoza status: %w", err)
		}

		if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubores.NamespaceName, chubores.OpenGyozaStatusType, chubores.OpenGyozaStatusID, resource.VersionUndefined)); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

func openGyozaConfigured(mc *config.MachineConfig) (bool, string, error) {
	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil {
		return false, "", nil
	}

	files, err := mc.Config().Machine().Files()
	if err != nil {
		return false, "", err
	}

	configured := false
	role := ""

	for _, f := range files {
		switch f.Path() {
		case openGyozaConfigPath:
			configured = true
		case openGyozaRolePath:
			role = strings.TrimSpace(f.Content())
		}
	}

	if configured && role == "" {
		role = openGyozaRoleServer
	}

	return configured, role, nil
}
