// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package opengyozaquorum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	roleServer       = "server"
	roleServerClient = "server-client"
	requestBodyLimit = 4096
)

var ErrUnsafeServerStop = errors.New("opengyoza server stop would break quorum")

func ReadRole(path string) (role string, configured bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}

		return "", false, err
	}

	return strings.TrimSpace(string(b)), true, nil
}

func IsServerRole(role string) bool {
	switch strings.TrimSpace(role) {
	case roleServer, roleServerClient:
		return true
	default:
		return false
	}
}

func CheckSafeServerStopFromPeers(peers []string) error {
	peerCount := len(peers)
	if peerCount <= 1 {
		return nil
	}

	quorum := (peerCount / 2) + 1
	remainingAfterStop := peerCount - 1

	if remainingAfterStop < quorum {
		return fmt.Errorf("%w: peers=%d quorum=%d", ErrUnsafeServerStop, peerCount, quorum)
	}

	return nil
}

func CheckSafeServerStop(ctx context.Context, client *http.Client, baseURL string) error {
	return CheckSafeServerStopWithToken(ctx, client, baseURL, "")
}

func CheckSafeServerStopWithToken(ctx context.Context, client *http.Client, baseURL, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/status/peers", nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))

		return fmt.Errorf("opengyoza peer check failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var peers []string

	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return fmt.Errorf("failed to decode opengyoza peers: %w", err)
	}

	return CheckSafeServerStopFromPeers(peers)
}
