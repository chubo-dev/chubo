// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var workloadStatusHTTPClient = &http.Client{
	Timeout: 2 * time.Second,
}

func fetchLeader(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := workloadStatusHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Both Nomad and Consul return JSON strings (quoted), but be tolerant of plain text.
	leader := strings.TrimSpace(string(body))
	leader = strings.Trim(leader, "\"")

	if leader == "" {
		return "", fmt.Errorf("empty leader response")
	}

	return leader, nil
}

func fetchPeers(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := workloadStatusHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var peers []string
	if err := json.Unmarshal(body, &peers); err != nil {
		return nil, fmt.Errorf("decode peers: %w", err)
	}

	return peers, nil
}

func queryOpenWontonStatus(ctx context.Context) (leader string, peerCount int32, lastErr error) {
	var errs []string

	leader, err := fetchLeader(ctx, "http://127.0.0.1:4646/v1/status/leader")
	if err != nil {
		errs = append(errs, "leader: "+err.Error())
	}

	peers, err := fetchPeers(ctx, "http://127.0.0.1:4646/v1/status/peers")
	if err != nil {
		errs = append(errs, "peers: "+err.Error())
	} else {
		peerCount = int32(len(peers))
	}

	if len(errs) > 0 {
		return leader, peerCount, fmt.Errorf(strings.Join(errs, "; "))
	}

	return leader, peerCount, nil
}

func queryOpenGyozaStatus(ctx context.Context) (leader string, peerCount int32, lastErr error) {
	var errs []string

	leader, err := fetchLeader(ctx, "http://127.0.0.1:8500/v1/status/leader")
	if err != nil {
		errs = append(errs, "leader: "+err.Error())
	}

	peers, err := fetchPeers(ctx, "http://127.0.0.1:8500/v1/status/peers")
	if err != nil {
		errs = append(errs, "peers: "+err.Error())
	} else {
		peerCount = int32(len(peers))
	}

	if len(errs) > 0 {
		return leader, peerCount, fmt.Errorf(strings.Join(errs, "; "))
	}

	return leader, peerCount, nil
}
