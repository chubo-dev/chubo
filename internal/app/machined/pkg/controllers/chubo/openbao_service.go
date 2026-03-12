// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	openBaoHostModePath    = "/var/lib/chubo/config/openbao.mode"
	openBaoHostConfigPath  = "/var/lib/chubo/config/openbao.hcl"
	openBaoModeHostService = "hostService"
	openBaoHTTPAddress     = "http://127.0.0.1:8200"
	openBaoQueryTimeout    = 5 * time.Second
	openBaoInitTimeout     = 30 * time.Second
	openBaoUnsealTimeout   = 10 * time.Second
)

var openBaoInitPath = "/var/lib/chubo/certs/openbao-init.json"

// OpenBaoServiceManager is the interface to v1alpha1 service manager.
type OpenBaoServiceManager interface {
	IsRunning(id string) (system.Service, bool, error)
	Load(services ...system.Service) []string
	Start(serviceIDs ...string) error
	Stop(ctx context.Context, serviceIDs ...string) error
}

// OpenBaoServiceController starts/stops and initializes the OS-managed OpenBao service.
type OpenBaoServiceController struct {
	V1Alpha1ServiceManager OpenBaoServiceManager
}

// Name implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Name() string {
	return "chubo.OpenBaoServiceController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Inputs() []controller.Input {
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
			ID:        optional.Some(services.OpenBaoServiceID),
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenBaoStatusType,
			Kind: controller.OutputExclusive,
		},
		{
			Type: secretres.OpenBaoInitType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
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

		configured, mode, err := openBaoConfigured(mc)
		if err != nil {
			return fmt.Errorf("error reading openbao intent: %w", err)
		}

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenBaoServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting openbao service state: %w", err)
		}

		if configured {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenBaoServiceID)
			if runErr != nil {
				ctrl.V1Alpha1ServiceManager.Load(&services.OpenBao{})
				running = false
			}

			if !running {
				if err := ctrl.V1Alpha1ServiceManager.Start(services.OpenBaoServiceID); err != nil {
					return fmt.Errorf("error starting openbao service: %w", err)
				}
			}
		} else {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenBaoServiceID)
			if runErr == nil && running {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.OpenBaoServiceID); err != nil {
					return fmt.Errorf("error stopping openbao service: %w", err)
				}
			}
		}

		running := false
		healthy := false

		if svcRes != nil {
			running = svcRes.TypedSpec().Running
			healthy = svcRes.TypedSpec().Healthy
		}

		initialized := false
		sealed := false
		lastError := ""

		if configured && healthy {
			initialized, sealed, lastError = ensureOpenBaoInitState(ctx, r)
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenBaoStatus(), func(res *chubores.OpenBaoStatus) error {
			res.TypedSpec().Configured = configured
			res.TypedSpec().Mode = mode
			res.TypedSpec().Running = running
			res.TypedSpec().Healthy = healthy
			res.TypedSpec().Initialized = initialized
			res.TypedSpec().Sealed = sealed
			res.TypedSpec().LastError = lastError

			return nil
		}); err != nil {
			return fmt.Errorf("error updating openbao status: %w", err)
		}

		if err := r.CleanupOutputs(
			ctx,
			resource.NewMetadata(chubores.NamespaceName, chubores.OpenBaoStatusType, chubores.OpenBaoStatusID, resource.VersionUndefined),
			resource.NewMetadata(secretres.NamespaceName, secretres.OpenBaoInitType, secretres.OpenBaoInitID, resource.VersionUndefined),
		); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

func openBaoConfigured(mc *config.MachineConfig) (bool, string, error) {
	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil {
		return false, "", nil
	}

	files, err := mc.Config().Machine().Files()
	if err != nil {
		return false, "", err
	}

	mode := ""
	configured := false

	for _, f := range files {
		switch f.Path() {
		case openBaoHostModePath:
			mode = strings.TrimSpace(f.Content())
		case openBaoHostConfigPath:
			configured = true
		}
	}

	if mode != openBaoModeHostService {
		return false, mode, nil
	}

	return configured, mode, nil
}

type openBaoInitStatus struct {
	Initialized bool `json:"initialized"`
}

type openBaoSealStatus struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
}

type openBaoInitResponse struct {
	RootToken  string   `json:"root_token"`
	KeysBase64 []string `json:"keys_base64"`
}

func ensureOpenBaoInitState(ctx context.Context, r controller.Runtime) (initialized bool, sealed bool, lastError string) {
	queryCtx, cancel := context.WithTimeout(ctx, openBaoQueryTimeout)
	status, err := queryOpenBaoSealStatus(queryCtx)
	cancel()
	if err != nil {
		return false, false, err.Error()
	}

	if !status.Initialized {
		initCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openBaoInitTimeout)
		resp, initErr := initOpenBao(initCtx)
		cancel()
		if initErr != nil {
			postInitCtx, cancel := context.WithTimeout(ctx, openBaoQueryTimeout)
			postInitStatus, postInitErr := queryOpenBaoSealStatus(postInitCtx)
			cancel()
			if postInitErr == nil && postInitStatus.Initialized {
				return true, postInitStatus.Sealed, "openbao initialized but init response was lost; wipe /var/lib/chubo/openbao to retry"
			}

			return false, true, initErr.Error()
		}

		if writeErr := persistOpenBaoInitState(ctx, r, resp); writeErr != nil {
			return false, true, writeErr.Error()
		}

		// Keep the legacy file write as a compatibility path for helper bundles,
		// but treat the sensitive runtime resource as the source of truth.
		if writeErr := persistOpenBaoInit(resp); writeErr != nil {
			lastError = writeErr.Error()
		}

		status.Initialized = true
		status.Sealed = true
	}

	if status.Sealed {
		initData, readErr := readOpenBaoInitState(ctx, r)
		if readErr != nil {
			return status.Initialized, true, readErr.Error()
		}

		if len(initData.KeysBase64) == 0 {
			return status.Initialized, true, "openbao init data has no unseal keys"
		}

		unsealCtx, cancel := context.WithTimeout(ctx, openBaoUnsealTimeout)
		err = unsealOpenBao(unsealCtx, initData.KeysBase64[0])
		cancel()
		if err != nil {
			return status.Initialized, true, err.Error()
		}

		queryCtx, cancel = context.WithTimeout(ctx, openBaoQueryTimeout)
		status, err = queryOpenBaoSealStatus(queryCtx)
		cancel()
		if err != nil {
			return true, true, err.Error()
		}
	}

	return status.Initialized, status.Sealed, ""
}

func persistOpenBaoInitState(ctx context.Context, r controller.Runtime, resp openBaoInitResponse) error {
	return safe.WriterModify(ctx, r, secretres.NewOpenBaoInit(), func(res *secretres.OpenBaoInit) error {
		res.TypedSpec().RootToken = strings.TrimSpace(resp.RootToken)
		res.TypedSpec().KeysBase64 = append([]string(nil), resp.KeysBase64...)

		return nil
	})
}

func readOpenBaoInitState(ctx context.Context, r controller.Runtime) (openBaoInitResponse, error) {
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

func queryOpenBaoSealStatus(ctx context.Context) (openBaoSealStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openBaoHTTPAddress+"/v1/sys/seal-status", nil)
	if err != nil {
		return openBaoSealStatus{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return openBaoSealStatus{}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return openBaoSealStatus{}, fmt.Errorf("openbao seal-status returned %s", resp.Status)
	}

	var out openBaoSealStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return openBaoSealStatus{}, err
	}

	return out, nil
}

func initOpenBao(ctx context.Context) (openBaoInitResponse, error) {
	body, err := json.Marshal(map[string]int{
		"secret_shares":    1,
		"secret_threshold": 1,
	})
	if err != nil {
		return openBaoInitResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, openBaoHTTPAddress+"/v1/sys/init", bytes.NewReader(body))
	if err != nil {
		return openBaoInitResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return openBaoInitResponse{}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return openBaoInitResponse{}, fmt.Errorf("openbao init returned %s", resp.Status)
	}

	var out openBaoInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return openBaoInitResponse{}, err
	}

	return out, nil
}

func unsealOpenBao(ctx context.Context, key string) error {
	body, err := json.Marshal(map[string]string{"key": key})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, openBaoHTTPAddress+"/v1/sys/unseal", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openbao unseal returned %s", resp.Status)
	}

	return nil
}

func persistOpenBaoInit(resp openBaoInitResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(openBaoInitPath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(openBaoInitPath, data, 0o600)
}

func readOpenBaoInit() (openBaoInitResponse, error) {
	data, err := os.ReadFile(openBaoInitPath)
	if err != nil {
		return openBaoInitResponse{}, err
	}

	var out openBaoInitResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return openBaoInitResponse{}, err
	}

	return out, nil
}
