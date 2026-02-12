// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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

const (
	openBaoModePath = "/var/lib/chubo/config/openbao.mode"
	openBaoJobPath  = "/var/lib/chubo/config/openbao.nomad.json"

	openBaoModeNomadJob = "nomadJob"
	openBaoDefaultJobID = "openbao"
	nomadHTTPAddress    = "http://127.0.0.1:4646"
)

var errNomadJobNotFound = errors.New("nomad job not found")

// OpenBaoJobStatusController ensures the OpenBao Nomad job exists and publishes reconciliation state.
type OpenBaoJobStatusController struct{}

// Name implements controller.Controller interface.
func (ctrl *OpenBaoJobStatusController) Name() string {
	return "chubo.OpenBaoJobStatusController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenBaoJobStatusController) Inputs() []controller.Input {
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
func (ctrl *OpenBaoJobStatusController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenBaoJobStatusType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenBaoJobStatusController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
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

		intent, err := readOpenBaoIntent(mc)
		if err != nil {
			return fmt.Errorf("error reading openbao intent: %w", err)
		}

		statusSpec := chubores.OpenBaoJobStatusSpec{
			Configured: intent.Configured,
			Mode:       intent.Mode,
			JobID:      intent.JobID,
		}

		if !intent.Configured {
			if err := ctrl.writeStatus(ctx, r, statusSpec); err != nil {
				return err
			}

			continue
		}

		nomadSvc, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenWontonServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting openwonton service state: %w", err)
		}

		if nomadSvc == nil || !nomadSvc.TypedSpec().Running || !nomadSvc.TypedSpec().Healthy {
			statusSpec.LastError = "openwonton service is not healthy"

			if err := ctrl.writeStatus(ctx, r, statusSpec); err != nil {
				return err
			}

			continue
		}

		reachable, present, reconcileErr := ensureOpenBaoNomadJob(ctx, intent.JobID, intent.NomadPayload)
		statusSpec.NomadReachable = reachable
		statusSpec.Present = present

		if reconcileErr != nil {
			statusSpec.LastError = reconcileErr.Error()
		}

		if err := ctrl.writeStatus(ctx, r, statusSpec); err != nil {
			return err
		}
	}
}

func (ctrl *OpenBaoJobStatusController) writeStatus(ctx context.Context, r controller.Runtime, spec chubores.OpenBaoJobStatusSpec) error {
	if err := safe.WriterModify(ctx, r, chubores.NewOpenBaoJobStatus(), func(res *chubores.OpenBaoJobStatus) error {
		res.TypedSpec().Configured = spec.Configured
		res.TypedSpec().Mode = spec.Mode
		res.TypedSpec().JobID = spec.JobID
		res.TypedSpec().NomadReachable = spec.NomadReachable
		res.TypedSpec().Present = spec.Present
		res.TypedSpec().LastError = spec.LastError

		return nil
	}); err != nil {
		return fmt.Errorf("error updating openbao job status: %w", err)
	}

	if err := r.CleanupOutputs(ctx, resource.NewMetadata(chubores.NamespaceName, chubores.OpenBaoJobStatusType, chubores.OpenBaoJobStatusID, resource.VersionUndefined)); err != nil {
		return fmt.Errorf("failed to cleanup outputs: %w", err)
	}

	return nil
}

type openBaoIntent struct {
	Configured   bool
	Mode         string
	JobID        string
	NomadPayload []byte
}

func readOpenBaoIntent(mc *config.MachineConfig) (openBaoIntent, error) {
	intent := openBaoIntent{
		JobID: openBaoDefaultJobID,
	}

	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil {
		return intent, nil
	}

	files, err := mc.Config().Machine().Files()
	if err != nil {
		return intent, err
	}

	for _, f := range files {
		switch f.Path() {
		case openBaoModePath:
			intent.Mode = strings.TrimSpace(f.Content())
		case openBaoJobPath:
			intent.Configured = true
			intent.NomadPayload = []byte(f.Content())
		}
	}

	if intent.Mode == "" && intent.Configured {
		intent.Mode = openBaoModeNomadJob
	}

	return intent, nil
}

func ensureOpenBaoNomadJob(ctx context.Context, jobID string, payload []byte) (reachable bool, present bool, err error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	present, err = nomadJobPresent(ctx, client, jobID)
	if err == nil {
		return true, present, nil
	}

	if !errors.Is(err, errNomadJobNotFound) {
		return false, false, err
	}

	if err := registerNomadJob(ctx, client, payload); err != nil {
		return true, false, err
	}

	// Re-check after registration so status is grounded in Nomad state.
	present, err = nomadJobPresent(ctx, client, jobID)
	if err == nil {
		return true, present, nil
	}

	if errors.Is(err, errNomadJobNotFound) {
		return true, false, nil
	}

	return true, false, err
}

func nomadJobPresent(ctx context.Context, client *http.Client, jobID string) (bool, error) {
	url := fmt.Sprintf("%s/v1/job/%s", nomadHTTPAddress, jobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close() //nolint:errcheck

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, errNomadJobNotFound
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return false, fmt.Errorf("nomad job lookup failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func registerNomadJob(ctx context.Context, client *http.Client, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, nomadHTTPAddress+"/v1/jobs", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	return fmt.Errorf("nomad job register failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}
