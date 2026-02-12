// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/chubo"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
)

const (
	bootstrapPayloadPath = "/var/lib/chubo/bootstrap/bootstrap.json"
	bootstrapSignerPath  = "/var/lib/chubo/bootstrap/signer.sha256"
)

// BootstrapStatusController reports the chubo bootstrap rendering state via COSI.
//
// This is intentionally lightweight: the "truth" is the rendered payload on disk, but we also
// report whether the applied machine config asked for bootstrap content (configured vs rendered).
type BootstrapStatusController struct{}

// Name implements controller.Controller interface.
func (ctrl *BootstrapStatusController) Name() string {
	return "chubo.BootstrapStatusController"
}

// Inputs implements controller.Controller interface.
func (ctrl *BootstrapStatusController) Inputs() []controller.Input {
	return []controller.Input{
		{
			Namespace: config.NamespaceName,
			Type:      config.MachineConfigType,
			ID:        optional.Some(config.ActiveID),
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *BootstrapStatusController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubo.BootstrapStatusType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *BootstrapStatusController) Run(ctx context.Context, r controller.Runtime, logger *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		r.StartTrackingOutputs()

		mc, err := safe.ReaderGetByID[*config.MachineConfig](ctx, r, config.ActiveID)
		if err != nil {
			if state.IsNotFoundError(err) {
				continue
			}

			return fmt.Errorf("error getting machine config: %w", err)
		}

		configured := false

		files, err := mc.Config().Machine().Files()
		if err != nil {
			// Treat as not configured, but keep this visible during debugging.
			logger.Debug("failed to read machine files from config", zap.Error(err))
		} else {
			for _, f := range files {
				if f.Path() == bootstrapPayloadPath {
					configured = true
					break
				}
			}
		}

		rendered := false

		if _, err := os.Stat(bootstrapPayloadPath); err == nil {
			rendered = true
		} else if !os.IsNotExist(err) {
			logger.Debug("failed to stat bootstrap payload", zap.Error(err))
		}

		signerSha := ""

		if b, err := os.ReadFile(bootstrapSignerPath); err == nil {
			signerSha = strings.TrimSpace(string(b))
		} else if !os.IsNotExist(err) {
			logger.Debug("failed to read bootstrap signer fingerprint", zap.Error(err))
		}

		if err := safe.WriterModify(ctx, r, chubo.NewBootstrapStatus(), func(res *chubo.BootstrapStatus) error {
			res.TypedSpec().Configured = configured
			res.TypedSpec().Rendered = rendered
			res.TypedSpec().SignerSha256 = signerSha

			return nil
		}); err != nil {
			return fmt.Errorf("error updating bootstrap status: %w", err)
		}

		if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubo.NamespaceName, chubo.BootstrapStatusType, chubo.BootstrapStatusID, resource.VersionUndefined)); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}
