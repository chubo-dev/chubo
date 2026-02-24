// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/services"
	chubores "github.com/chubo-dev/chubo/pkg/machinery/resources/chubo"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	v1alpha1res "github.com/chubo-dev/chubo/pkg/machinery/resources/v1alpha1"
)

var (
	openGyozaBootstrapExpectRe = regexp.MustCompile(`(?m)^\s*bootstrap_expect\s*=\s*(\d+)\s*$`)
	openGyozaRetryJoinRe       = regexp.MustCompile(`(?m)^\s*retry_join\s*=\s*(\[[^\n]*\])\s*$`)
)

// OpenGyozaBootstrapStatusController performs best-effort bootstrap checks and publishes state via COSI.
type OpenGyozaBootstrapStatusController struct{}

// Name implements controller.Controller interface.
func (ctrl *OpenGyozaBootstrapStatusController) Name() string {
	return "chubo.OpenGyozaBootstrapStatusController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenGyozaBootstrapStatusController) Inputs() []controller.Input {
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
func (ctrl *OpenGyozaBootstrapStatusController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenGyozaBootstrapStatusType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenGyozaBootstrapStatusController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		case <-ticker.C:
		}

		r.StartTrackingOutputs()

		mc, err := safe.ReaderGetByID[*config.MachineConfig](ctx, r, config.ActiveID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting machine config: %w", err)
		}

		desired, derr := readOpenGyozaDesiredState(mc)

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenGyozaServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting opengyoza service state: %w", err)
		}

		running := false
		healthy := false
		if svcRes != nil {
			running = svcRes.TypedSpec().Running
			healthy = svcRes.TypedSpec().Healthy
		}

		leader := ""
		peerCount := int32(0)
		clusterReady := false
		lastError := ""
		aclReady := false
		aclLastError := ""
		aclTokenSHA := ""

		if derr != nil {
			lastError = derr.Error()
		}

		if desired.Configured {
			token := deriveWorkloadACLTokenFromMachineConfig(mc, "consul")
			if strings.TrimSpace(token) != "" {
				sum := sha256.Sum256([]byte(token))
				aclTokenSHA = hex.EncodeToString(sum[:])
			}

			if healthy && derr == nil {
				qctx, cancel := context.WithTimeout(ctx, 2*time.Second)

				client, err := services.NewChuboServiceHTTPClient(services.OpenGyozaServiceID, 2*time.Second)
				if err != nil {
					aclLastError = err.Error()
				} else {
					aclReady, err = ensureConsulACL(qctx, client, openGyozaHTTPAddress, token, isOpenGyozaServerRole(desired.Role))
					if err != nil {
						aclLastError = err.Error()
					}
				}

				if aclReady {
					leader, peerCount, err = queryOpenGyozaStatus(qctx, token)
					if err != nil && lastError == "" {
						lastError = err.Error()
					}

					switch {
					case isOpenGyozaServerRole(desired.Role):
						clusterReady = leader != "" && peerCount >= desired.BootstrapExpect
					default:
						clusterReady = leader != ""
					}
				}

				cancel()
			}
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenGyozaBootstrapStatus(), func(res *chubores.OpenGyozaBootstrapStatus) error {
			res.TypedSpec().Configured = desired.Configured
			res.TypedSpec().Role = desired.Role
			res.TypedSpec().BootstrapExpect = desired.BootstrapExpect
			res.TypedSpec().Join = desired.Join
			res.TypedSpec().Running = running
			res.TypedSpec().Healthy = healthy
			res.TypedSpec().ACLReady = aclReady
			res.TypedSpec().ACLLastError = aclLastError
			res.TypedSpec().Leader = leader
			res.TypedSpec().PeerCount = peerCount
			res.TypedSpec().ClusterReady = clusterReady
			res.TypedSpec().LastError = lastError
			res.TypedSpec().ACLTokenSHA256 = aclTokenSHA

			return nil
		}); err != nil {
			return fmt.Errorf("error updating opengyoza bootstrap status: %w", err)
		}

		if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubores.NamespaceName, chubores.OpenGyozaBootstrapStatusType, chubores.OpenGyozaBootstrapStatusID, resource.VersionUndefined)); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

type openGyozaDesiredState struct {
	Configured      bool
	Role            string
	BootstrapExpect int32
	Join            []string
}

func readOpenGyozaDesiredState(mc *config.MachineConfig) (openGyozaDesiredState, error) {
	var desired openGyozaDesiredState

	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil {
		return desired, nil
	}

	files, err := mc.Config().Machine().Files()
	if err != nil {
		return desired, err
	}

	var cfgRaw string

	for _, f := range files {
		switch f.Path() {
		case openGyozaConfigPath:
			desired.Configured = true
			cfgRaw = f.Content()
		case openGyozaRolePath:
			desired.Role = strings.TrimSpace(f.Content())
		}
	}

	if !desired.Configured {
		return desired, nil
	}

	if desired.Role == "" {
		desired.Role = openGyozaRoleServer
	}

	if strings.TrimSpace(cfgRaw) == "" {
		return desired, fmt.Errorf("opengyoza config is empty")
	}

	expect, err := parseBootstrapExpect(openGyozaBootstrapExpectRe, cfgRaw)
	if err != nil {
		return desired, fmt.Errorf("parse opengyoza bootstrap_expect: %w", err)
	}

	desired.BootstrapExpect = int32(expect)

	join, err := parseRetryJoin(openGyozaRetryJoinRe, cfgRaw)
	if err != nil {
		return desired, fmt.Errorf("parse opengyoza retry_join: %w", err)
	}

	desired.Join = join

	return desired, nil
}
