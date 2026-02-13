// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	chuboacl "github.com/chubo-dev/chubo/pkg/chubo/acl"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
)

const (
	nomadTokenHeader  = "X-Nomad-Token"
	consulTokenHeader = "X-Consul-Token"

	requestBodyLimit = 4096
)

var errMissingACLToken = errors.New("missing derived ACL token")

type bootstrapSecretRequest struct {
	BootstrapSecret string `json:"BootstrapSecret"`
}

func deriveWorkloadACLTokenFromMachineConfig(mc *config.MachineConfig, name string) string {
	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil || mc.Config().Machine().Security() == nil {
		return ""
	}

	return chuboacl.WorkloadToken(mc.Config().Machine().Security().Token(), name)
}

func ensureNomadACL(ctx context.Context, client *http.Client, baseURL, token string, allowBootstrap bool) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, errMissingACLToken
	}

	ok, selfErr := tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", nomadTokenHeader, token)
	if ok {
		return true, nil
	}

	if !allowBootstrap {
		return false, selfErr
	}

	if err := bootstrapACL(ctx, client, http.MethodPost, baseURL+"/v1/acl/bootstrap", token); err != nil {
		return false, err
	}

	ok, err := tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", nomadTokenHeader, token)
	if ok {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return false, fmt.Errorf("nomad ACL token not accepted after bootstrap")
}

func ensureConsulACL(ctx context.Context, client *http.Client, baseURL, token string, allowBootstrap bool) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, errMissingACLToken
	}

	ok, selfErr := tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", consulTokenHeader, token)
	if ok {
		return true, nil
	}

	if !allowBootstrap {
		return false, selfErr
	}

	if err := bootstrapACL(ctx, client, http.MethodPut, baseURL+"/v1/acl/bootstrap", token); err != nil {
		return false, err
	}

	ok, err := tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", consulTokenHeader, token)
	if ok {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return false, fmt.Errorf("consul ACL token not accepted after bootstrap")
}

func tokenSelf(ctx context.Context, client *http.Client, url, header, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set(header, token)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))

	return false, fmt.Errorf("token self failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func bootstrapACL(ctx context.Context, client *http.Client, method, url, token string) error {
	payload, err := json.Marshal(bootstrapSecretRequest{BootstrapSecret: token})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))
	bodyStr := strings.TrimSpace(string(body))
	bodyLower := strings.ToLower(bodyStr)

	// Treat "already bootstrapped" (or similar) as success to make the operation idempotent.
	if strings.Contains(bodyLower, "bootstrapp") && strings.Contains(bodyLower, "already") {
		return nil
	}

	return fmt.Errorf("ACL bootstrap failed: status=%d body=%s", resp.StatusCode, bodyStr)
}
