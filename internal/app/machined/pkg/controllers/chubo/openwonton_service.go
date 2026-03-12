// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/services"
	chubores "github.com/chubo-dev/chubo/pkg/machinery/resources/chubo"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	secretres "github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
	v1alpha1res "github.com/chubo-dev/chubo/pkg/machinery/resources/v1alpha1"
)

const (
	openWontonConfigPath  = "/var/lib/chubo/config/openwonton.hcl"
	openWontonRolePath    = "/var/lib/chubo/config/openwonton.role"
	openWontonBinaryPath  = "/var/lib/chubo/bin/openwonton"
	openWontonFallback    = "/usr/bin/init"
	openWontonRoleServer  = "server"
	openWontonRoleHybrid  = "server-client"
	openWontonHTTPAddress = "https://127.0.0.1:4646"
)

func isOpenWontonServerRole(role string) bool {
	switch strings.TrimSpace(role) {
	case openWontonRoleServer, openWontonRoleHybrid:
		return true
	default:
		return false
	}
}

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

		configured, role, err := openWontonConfigured(mc)
		if err != nil {
			return fmt.Errorf("error reading openwonton intent: %w", err)
		}

		hostOpenBaoReady := true
		hostOpenBaoErr := ""

		if configured && isOpenWontonServerRole(role) {
			hostOpenBaoReady, hostOpenBaoErr = prepareOpenWontonHostOpenBao(ctx, r, mc)
		}

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenWontonServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting openwonton service state: %w", err)
		}

		if configured {
			_, dockerRunning, dockerErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.DockerServiceID)
			if dockerErr != nil {
				ctrl.V1Alpha1ServiceManager.Load(&services.Docker{})
				dockerRunning = false
			}

			if !dockerRunning {
				if err := ctrl.V1Alpha1ServiceManager.Start(services.DockerServiceID); err != nil {
					return fmt.Errorf("error starting docker service: %w", err)
				}
			}

			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenWontonServiceID)
			if runErr != nil {
				ctrl.V1Alpha1ServiceManager.Load(&services.OpenWonton{})
				running = false
			}

			if !running && hostOpenBaoReady {
				if err := ctrl.V1Alpha1ServiceManager.Start(services.OpenWontonServiceID); err != nil {
					return fmt.Errorf("error starting openwonton service: %w", err)
				}
			}

			if !hostOpenBaoReady && runErr == nil && running {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.OpenWontonServiceID); err != nil {
					return fmt.Errorf("error stopping openwonton service while waiting for openbao token: %w", err)
				}
			}
		} else {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenWontonServiceID)
			if runErr == nil && running {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.OpenWontonServiceID); err != nil {
					return fmt.Errorf("error stopping openwonton service: %w", err)
				}
			}

			_, dockerRunning, dockerErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.DockerServiceID)
			if dockerErr == nil && dockerRunning {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.DockerServiceID); err != nil {
					return fmt.Errorf("error stopping docker service: %w", err)
				}
			}
		}

		running := false
		healthy := false

		if svcRes != nil {
			running = svcRes.TypedSpec().Running
			healthy = svcRes.TypedSpec().Healthy
		}

		leader := ""
		peerCount := int32(0)
		lastError := ""
		aclReady := false
		aclLastError := ""

		if configured && !hostOpenBaoReady {
			lastError = hostOpenBaoErr
		}

		if configured && healthy {
			token := deriveWorkloadACLTokenFromMachineConfig(mc, "nomad")

			qctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			var qerr error

			client, err := services.NewChuboServiceHTTPClient(services.OpenWontonServiceID, 2*time.Second)
			if err != nil {
				aclLastError = err.Error()
			} else {
				aclReady, qerr = ensureNomadACL(qctx, client, openWontonHTTPAddress, token, isOpenWontonServerRole(role))
				if qerr != nil {
					aclLastError = qerr.Error()
				}
			}

			leader, peerCount, qerr = queryOpenWontonStatus(qctx, token)
			cancel()
			if qerr != nil {
				lastError = qerr.Error()
			}
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenWontonStatus(), func(res *chubores.OpenWontonStatus) error {
			res.TypedSpec().Configured = configured
			res.TypedSpec().Role = role
			res.TypedSpec().Running = running
			res.TypedSpec().Healthy = healthy
			res.TypedSpec().BinaryMode = detectServiceBinaryMode(openWontonBinaryPath, openWontonFallback)
			res.TypedSpec().Leader = leader
			res.TypedSpec().PeerCount = peerCount
			res.TypedSpec().LastError = lastError
			res.TypedSpec().ACLReady = aclReady
			res.TypedSpec().ACLLastError = aclLastError

			return nil
		}); err != nil {
			return fmt.Errorf("error updating openwonton status: %w", err)
		}

		if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubores.NamespaceName, chubores.OpenWontonStatusType, chubores.OpenWontonStatusID, resource.VersionUndefined)); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

func prepareOpenWontonHostOpenBao(ctx context.Context, r controller.Runtime, mc *config.MachineConfig) (bool, string) {
	configured, mode, err := openBaoConfigured(mc)
	if err != nil {
		return false, fmt.Sprintf("error reading openbao intent: %v", err)
	}

	if !configured || mode != openBaoModeHostService {
		return true, ""
	}

	initData, err := readOpenBaoInitFromRuntime(ctx, r)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "waiting for host-native openbao init material"
		}

		return false, fmt.Sprintf("error reading openbao init material: %v", err)
	}

	token := strings.TrimSpace(initData.RootToken)
	if token == "" {
		return false, "host-native openbao root token is empty"
	}

	if err := ensureOpenWontonVaultToken(token); err != nil {
		return false, fmt.Sprintf("error updating openwonton vault token: %v", err)
	}

	return true, ""
}

func readOpenBaoInitFromRuntime(ctx context.Context, r controller.Runtime) (openBaoInitResponse, error) {
	initRes, err := safe.ReaderGetByID[*secretres.OpenBaoInit](ctx, r, secretres.OpenBaoInitID)
	if err == nil {
		out := openBaoInitResponse{
			RootToken:  strings.TrimSpace(initRes.TypedSpec().RootToken),
			KeysBase64: append([]string(nil), initRes.TypedSpec().KeysBase64...),
		}

		if out.RootToken != "" || len(out.KeysBase64) > 0 {
			return out, nil
		}
	}

	return readOpenBaoInit()
}

func ensureOpenWontonVaultToken(token string) error {
	data, err := os.ReadFile(openWontonConfigPath)
	if err != nil {
		return err
	}

	updated, changed, err := injectOpenWontonVaultToken(string(data), token)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	return os.WriteFile(openWontonConfigPath, []byte(updated), 0o600)
}

func injectOpenWontonVaultToken(cfg string, token string) (string, bool, error) {
	const blockStart = "vault {\n"

	start := strings.Index(cfg, blockStart)
	if start == -1 {
		return cfg, false, nil
	}

	rest := cfg[start:]
	endRel := strings.Index(rest, "\n}\n")
	if endRel == -1 {
		return cfg, false, fmt.Errorf("vault block is malformed")
	}

	end := start + endRel + len("\n}")
	block := cfg[start:end]
	lines := strings.Split(block, "\n")
	tokenLine := fmt.Sprintf("  token = %q", token)

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "token = ") {
			if line == tokenLine {
				return cfg, false, nil
			}

			lines[i] = tokenLine

			newBlock := strings.Join(lines, "\n")
			return cfg[:start] + newBlock + cfg[end:], true, nil
		}
	}

	if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) != "}" {
		return cfg, false, fmt.Errorf("vault block is malformed")
	}

	lines = append(lines[:len(lines)-1], tokenLine, lines[len(lines)-1])
	newBlock := strings.Join(lines, "\n")

	return cfg[:start] + newBlock + cfg[end:], true, nil
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
