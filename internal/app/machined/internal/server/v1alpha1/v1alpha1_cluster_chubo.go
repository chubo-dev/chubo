// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package runtime provides the runtime implementation.
package runtime

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	clusterapi "github.com/chubo-dev/chubo/pkg/machinery/api/cluster"
)

// HealthCheck implements the cluster.ClusterServer interface.
//
// Chubo-OS intentionally disables Kubernetes-oriented cluster checks.
func (s *Server) HealthCheck(_ *clusterapi.HealthCheckRequest, _ clusterapi.ClusterService_HealthCheckServer) error {
	return status.Error(codes.Unimplemented, "cluster health checks are not available in chubo mode")
}
