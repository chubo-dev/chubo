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
	"google.golang.org/protobuf/types/known/emptypb"

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

func TestKubeconfigRPCDisabledInChuboMode(t *testing.T) {
	t.Parallel()

	srv := &Server{}

	err := srv.Kubeconfig(&emptypb.Empty{}, nil)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, st.Code())
	require.Contains(t, st.Message(), "chubo mode")
	require.Contains(t, st.Message(), "NomadConfig")
	require.Contains(t, st.Message(), "ConsulConfig")
	require.Contains(t, st.Message(), "OpenBaoConfig")
}
