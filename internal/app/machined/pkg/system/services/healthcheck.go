// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func simpleHealthCheck(ctx context.Context, url string) error {
	return simpleHealthCheckWithClient(ctx, url, http.DefaultClient)
}

func simpleHealthCheckWithClient(ctx context.Context, url string, client *http.Client) error {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Keep the error readable in logs while still providing some context.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10)) //nolint:errcheck
		return fmt.Errorf("health check %q: %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}

	return nil
}
