// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package v1alpha1

import (
	"context"

	"github.com/cosi-project/runtime/pkg/state"

	"github.com/chubo-dev/chubo/pkg/machinery/config/validation"
)

// These methods are only used in `chubo` builds, but the call sites live in
// upstream validation code paths which must compile in all build modes.
func (c *Config) validateChuboOS(validation.RuntimeMode, ...validation.Option) ([]string, error) {
	return nil, nil
}

func (c *Config) runtimeValidateChuboOS(context.Context, state.State, validation.RuntimeMode, ...validation.Option) ([]string, error) {
	return nil, nil
}
