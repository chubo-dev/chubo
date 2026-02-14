// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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
	openWontonBootstrapExpectRe = regexp.MustCompile(`(?m)^\s*bootstrap_expect\s*=\s*(\d+)\s*$`)
	openWontonRetryJoinRe       = regexp.MustCompile(`(?m)^\s*retry_join\s*=\s*(\[[^\n]*\])\s*$`)
)

// OpenWontonBootstrapStatusController performs best-effort bootstrap checks and publishes state via COSI.
type OpenWontonBootstrapStatusController struct{}

// Name implements controller.Controller interface.
func (ctrl *OpenWontonBootstrapStatusController) Name() string {
	return "chubo.OpenWontonBootstrapStatusController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenWontonBootstrapStatusController) Inputs() []controller.Input {
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
func (ctrl *OpenWontonBootstrapStatusController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenWontonBootstrapStatusType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenWontonBootstrapStatusController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
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

		desired, derr := readOpenWontonDesiredState(mc)

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenWontonServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting openwonton service state: %w", err)
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
			token := deriveWorkloadACLTokenFromMachineConfig(mc, "nomad")
			if strings.TrimSpace(token) != "" {
				sum := sha256.Sum256([]byte(token))
				aclTokenSHA = hex.EncodeToString(sum[:])
			}

			if healthy && derr == nil {
				qctx, cancel := context.WithTimeout(ctx, 2*time.Second)

				client, err := services.NewChuboServiceHTTPClient(services.OpenWontonServiceID, 2*time.Second)
				if err != nil {
					aclLastError = err.Error()
				} else {
					aclReady, err = ensureNomadACL(qctx, client, openWontonHTTPAddress, token, desired.Role == openWontonRoleServer)
					if err != nil {
						aclLastError = err.Error()
					}
				}

				if aclReady {
					leader, peerCount, err = queryOpenWontonStatus(qctx, token)
					if err != nil && lastError == "" {
						lastError = err.Error()
					}

					switch desired.Role {
					case openWontonRoleServer:
						clusterReady = leader != "" && peerCount >= desired.BootstrapExpect
					default:
						clusterReady = leader != ""
					}
				}

				cancel()
			}
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenWontonBootstrapStatus(), func(res *chubores.OpenWontonBootstrapStatus) error {
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
			return fmt.Errorf("error updating openwonton bootstrap status: %w", err)
		}

		if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubores.NamespaceName, chubores.OpenWontonBootstrapStatusType, chubores.OpenWontonBootstrapStatusID, resource.VersionUndefined)); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

type openWontonDesiredState struct {
	Configured      bool
	Role            string
	BootstrapExpect int32
	Join            []string
}

func readOpenWontonDesiredState(mc *config.MachineConfig) (openWontonDesiredState, error) {
	var desired openWontonDesiredState

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
		case openWontonConfigPath:
			desired.Configured = true
			cfgRaw = f.Content()
		case openWontonRolePath:
			desired.Role = strings.TrimSpace(f.Content())
		}
	}

	if !desired.Configured {
		return desired, nil
	}

	if desired.Role == "" {
		desired.Role = openWontonRoleServer
	}

	if strings.TrimSpace(cfgRaw) == "" {
		return desired, fmt.Errorf("openwonton config is empty")
	}

	expect, err := parseBootstrapExpect(openWontonBootstrapExpectRe, cfgRaw)
	if err != nil {
		return desired, fmt.Errorf("parse openwonton bootstrap_expect: %w", err)
	}

	desired.BootstrapExpect = int32(expect)

	join, err := parseRetryJoin(openWontonRetryJoinRe, cfgRaw)
	if err != nil {
		return desired, fmt.Errorf("parse openwonton retry_join: %w", err)
	}

	desired.Join = join

	return desired, nil
}

func parseBootstrapExpect(re *regexp.Regexp, raw string) (int, error) {
	m := re.FindStringSubmatch(raw)
	if len(m) != 2 {
		return 0, fmt.Errorf("bootstrap_expect not found")
	}

	v, err := strconv.Atoi(m[1])
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid value %q", m[1])
	}

	return v, nil
}

func parseRetryJoin(re *regexp.Regexp, raw string) ([]string, error) {
	m := re.FindStringSubmatch(raw)
	if len(m) != 2 {
		return nil, nil
	}

	var join []string
	if err := json.Unmarshal([]byte(m[1]), &join); err != nil {
		return nil, err
	}

	for i := range join {
		join[i] = strings.TrimSpace(join[i])
	}

	return join, nil
}
