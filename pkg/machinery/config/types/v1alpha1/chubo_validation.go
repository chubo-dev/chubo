// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package v1alpha1

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosi-project/runtime/pkg/state"
	"github.com/hashicorp/go-multierror"

	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/config/validation"
)

func (c *Config) validateChuboOS(mode validation.RuntimeMode, options ...validation.Option) ([]string, error) {
	var (
		warnings []string
		result   *multierror.Error
	)

	if c.MachineConfig == nil {
		result = multierror.Append(result, errors.New("machine instructions are required"))
		return nil, result.ErrorOrNil()
	}

	// `chubo` relies on OS API + trustd on every node; keep the Talos "controlplane"
	// machine type as the compatibility signal.
	if !c.Machine().Type().IsControlPlane() {
		result = multierror.Append(result, fmt.Errorf("chubo requires machine.type to be %q (or %q)", machine.TypeControlPlane.String(), machine.TypeInit.String()))
	}

	// Trust: keep Talos trustd token + OS issuing CA (for now).
	if c.Machine().Security().Token() == "" {
		result = multierror.Append(result, errors.New("trustd token is required (.machine.token)"))
	}

	issuingCA := c.Machine().Security().IssuingCA()
	acceptedCAs := c.Machine().Security().AcceptedCAs()

	if issuingCA == nil && len(acceptedCAs) == 0 {
		result = multierror.Append(result, errors.New("issuing CA or some accepted CAs are required (.machine.ca, machine.acceptedCAs)"))
	}

	// trustd needs the issuing CA private key to mint the API server certificate.
	if issuingCA == nil || len(issuingCA.Key) == 0 {
		result = multierror.Append(result, errors.New("issuing CA key is required for OS API trustd flows (.machine.ca.key)"))
	}

	// Install requirements (keep parity with upstream checks).
	if mode.RequiresInstall() {
		if c.MachineConfig.MachineInstall == nil {
			result = multierror.Append(result, fmt.Errorf("install instructions are required in %q mode", mode))
		} else {
			matcher, err := c.MachineConfig.MachineInstall.DiskMatchExpression()
			if err != nil {
				result = multierror.Append(result, fmt.Errorf("install disk selector is invalid: %w", err))
			}

			if c.MachineConfig.MachineInstall.InstallDisk == "" && matcher == nil {
				result = multierror.Append(result, errors.New("either install disk or diskSelector should be defined"))
			}

			if len(c.MachineConfig.MachineInstall.InstallExtraKernelArgs) > 0 && c.MachineConfig.MachineInstall.GrubUseUKICmdline() {
				result = multierror.Append(result, errors.New("install.extraKernelArgs and install.grubUseUKICmdline can't be used together"))
			}
		}
	}

	_ = options // reserved for future strict mode handling

	return warnings, result.ErrorOrNil()
}

func (c *Config) runtimeValidateChuboOS(ctx context.Context, st state.State, mode validation.RuntimeMode, opt ...validation.Option) ([]string, error) {
	// For Phase 2 we keep runtime validation minimal: reuse the static validator.
	// Disk selector matching and deeper runtime checks can be added once the schema stabilizes.
	return c.validateChuboOS(mode, opt...)
}
