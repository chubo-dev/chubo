// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build integration_api && integration_k8s

package api

import (
	"context"
	"time"

	"github.com/chubo-dev/chubo/internal/integration/base"
	"github.com/chubo-dev/chubo/pkg/images"
	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/client"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

// ContainersSuite ...
type ContainersSuite struct {
	base.APISuite

	ctx       context.Context //nolint:containedctx
	ctxCancel context.CancelFunc
}

// SuiteName ...
func (suite *ContainersSuite) SuiteName() string {
	return "api.ContainersSuite"
}

// SetupTest ...
func (suite *ContainersSuite) SetupTest() {
	suite.ctx, suite.ctxCancel = context.WithTimeout(context.Background(), time.Minute)
}

// TearDownTest ...
func (suite *ContainersSuite) TearDownTest() {
	if suite.ctxCancel != nil {
		suite.ctxCancel()
	}
}

// TestSandboxImage verifies sandbox image.
func (suite *ContainersSuite) TestSandboxImage() {
	node := suite.RandomDiscoveredNodeInternalIP(machine.TypeControlPlane)
	ctx := client.WithNode(suite.ctx, node)

	resp, err := suite.Client.Containers(ctx, constants.WorkloadContainerdNamespace, common.ContainerDriver_CRI)
	suite.Require().NoError(err)

	suite.Assert().NotEmpty(resp.GetMessages())

	for _, message := range resp.GetMessages() {
		suite.Assert().NotEmpty(message.GetContainers())

		matched := false

		for _, ctr := range message.GetContainers() {
			if ctr.PodId == ctr.Id {
				suite.Assert().Equal(images.DefaultSandboxImage, ctr.Image)

				matched = true
			}
		}

		suite.Assert().True(matched, "no pods found, node %s", node)
	}
}

func init() {
	allSuites = append(allSuites, new(ContainersSuite))
}
