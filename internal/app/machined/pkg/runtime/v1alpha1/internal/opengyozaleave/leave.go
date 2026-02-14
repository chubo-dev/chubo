// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package opengyozaleave

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const requestBodyLimit = 4096

func Leave(ctx context.Context, client *http.Client, baseURL string) error {
	return LeaveWithToken(ctx, client, baseURL, "")
}

func LeaveWithToken(ctx context.Context, client *http.Client, baseURL, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(baseURL, "/")+"/v1/agent/leave", nil)
	if err != nil {
		return err
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set("X-Consul-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))

	return fmt.Errorf("opengyoza leave failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}
