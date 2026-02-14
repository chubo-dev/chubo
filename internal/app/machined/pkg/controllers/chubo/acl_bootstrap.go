// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"net/http"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/chubo/internal/aclbootstrap"
	chuboacl "github.com/chubo-dev/chubo/pkg/chubo/acl"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
)

func deriveWorkloadACLTokenFromMachineConfig(mc *config.MachineConfig, name string) string {
	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil || mc.Config().Machine().Security() == nil {
		return ""
	}

	return chuboacl.WorkloadToken(mc.Config().Machine().Security().Token(), name)
}

func ensureNomadACL(ctx context.Context, client *http.Client, baseURL, token string, allowBootstrap bool) (bool, error) {
	return aclbootstrap.EnsureNomadACL(ctx, client, baseURL, token, allowBootstrap)
}

func ensureConsulACL(ctx context.Context, client *http.Client, baseURL, token string, allowBootstrap bool) (bool, error) {
	return aclbootstrap.EnsureConsulACL(ctx, client, baseURL, token, allowBootstrap)
}
