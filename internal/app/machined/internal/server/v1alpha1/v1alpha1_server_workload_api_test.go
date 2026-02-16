// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
)

func TestBootstrapRPCDisabledInChuboMode(t *testing.T) {
	t.Parallel()

	srv := &Server{}

	_, err := srv.Bootstrap(context.Background(), &machine.BootstrapRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, st.Code())
	require.Contains(t, st.Message(), "chubo mode")
	require.Contains(t, st.Message(), "openwontonbootstrapstatus")
}
