// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"context"

	"github.com/cosi-project/runtime/pkg/state"

	"github.com/chubo-dev/chubo/pkg/machinery/config/validation"
)

// Validate implements the config.Provider interface.
func (c *Config) Validate(mode validation.RuntimeMode, options ...validation.Option) ([]string, error) {
	return c.validateChuboOS(mode, options...)
}

// RuntimeValidate implements the config.Provider interface.
func (c *Config) RuntimeValidate(ctx context.Context, st state.State, mode validation.RuntimeMode, opt ...validation.Option) ([]string, error) {
	return c.runtimeValidateChuboOS(ctx, st, mode, opt...)
}
