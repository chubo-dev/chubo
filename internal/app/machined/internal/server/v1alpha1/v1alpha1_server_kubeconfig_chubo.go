// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package runtime

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
)

// Kubeconfig is intentionally disabled in chubo mode.
func (s *Server) Kubeconfig(_ *emptypb.Empty, _ machine.MachineService_KubeconfigServer) error {
	return status.Error(codes.Unimplemented, "kubeconfig is not available in chubo mode")
}
