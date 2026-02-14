// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package diagnostics

import (
	"context"

	"github.com/cosi-project/runtime/pkg/controller"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/runtime"
)

// KubeletCSRNotApprovedCheck is disabled in chubo mode.
func KubeletCSRNotApprovedCheck(context.Context, controller.Reader, *zap.Logger) (*runtime.DiagnosticSpec, error) {
	return nil, nil
}
