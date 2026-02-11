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
	openWontonConfigPath = "/var/lib/chubo/config/openwonton.hcl"
	openWontonRolePath   = "/var/lib/chubo/config/openwonton.role"
	openWontonRoleServer = "server"
)

// OpenWontonServiceManager is the interface to v1alpha1 service manager.
type OpenWontonServiceManager interface {
	IsRunning(id string) (system.Service, bool, error)
	Load(services ...system.Service) []string
	Start(serviceIDs ...string) error
	Stop(ctx context.Context, serviceIDs ...string) error
}

// OpenWontonServiceController starts/stops the OS-managed openwonton service from machine config intent.
type OpenWontonServiceController struct {
	V1Alpha1ServiceManager OpenWontonServiceManager
}

// Name implements controller.Controller interface.
func (ctrl *OpenWontonServiceController) Name() string {
	return "chubo.OpenWontonServiceController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenWontonServiceController) Inputs() []controller.Input {
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
			ID:        optional.Some(services.OpenWontonServiceID),
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *OpenWontonServiceController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenWontonStatusType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenWontonServiceController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
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

		configured, role, err := openWontonConfigured(mc)
		if err != nil {
			return fmt.Errorf("error reading openwonton intent: %w", err)
		}

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenWontonServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting openwonton service state: %w", err)
		}

		if configured {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenWontonServiceID)
			if runErr != nil {
				ctrl.V1Alpha1ServiceManager.Load(&services.OpenWonton{})
				running = false
			}

			if !running {
				if err := ctrl.V1Alpha1ServiceManager.Start(services.OpenWontonServiceID); err != nil {
					return fmt.Errorf("error starting openwonton service: %w", err)
				}
			}
		} else {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenWontonServiceID)
			if runErr == nil && running {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.OpenWontonServiceID); err != nil {
					return fmt.Errorf("error stopping openwonton service: %w", err)
				}
			}
		}

		running := false
		healthy := false

		if svcRes != nil {
			running = svcRes.TypedSpec().Running
			healthy = svcRes.TypedSpec().Healthy
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenWontonStatus(), func(res *chubores.OpenWontonStatus) error {
			res.TypedSpec().Configured = configured
			res.TypedSpec().Role = role
			res.TypedSpec().Running = running
			res.TypedSpec().Healthy = healthy

			return nil
		}); err != nil {
			return fmt.Errorf("error updating openwonton status: %w", err)
		}

		if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubores.NamespaceName, chubores.OpenWontonStatusType, chubores.OpenWontonStatusID, resource.VersionUndefined)); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

func openWontonConfigured(mc *config.MachineConfig) (bool, string, error) {
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
		case openWontonConfigPath:
			configured = true
		case openWontonRolePath:
			role = strings.TrimSpace(f.Content())
		}
	}

	if configured && role == "" {
		role = openWontonRoleServer
	}

	return configured, role, nil
}
