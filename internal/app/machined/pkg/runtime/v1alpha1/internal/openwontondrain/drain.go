// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package openwontondrain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	roleClient       = "client"
	roleServerClient = "server-client"
	requestBodyLimit = 4096
)

var ErrNodeNotFound = errors.New("openwonton node not found")

type nodeRecord struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

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

func IsClientRole(role string) bool {
	switch strings.TrimSpace(role) {
	case roleClient, roleServerClient:
		return true
	default:
		return false
	}
}

func DrainNode(ctx context.Context, client *http.Client, baseURL, nodeName string, deadline time.Duration) error {
	return DrainNodeWithToken(ctx, client, baseURL, nodeName, deadline, "")
}

func DrainNodeWithToken(ctx context.Context, client *http.Client, baseURL, nodeName string, deadline time.Duration, token string) error {
	nodeID, err := resolveNodeID(ctx, client, baseURL, nodeName, token)
	if err != nil {
		return err
	}

	if err := setNodeEligibility(ctx, client, baseURL, nodeID, "ineligible", token); err != nil {
		return err
	}

	return enableNodeDrain(ctx, client, baseURL, nodeID, deadline, token)
}

func resolveNodeID(ctx context.Context, client *http.Client, baseURL, nodeName, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/nodes", nil)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set("X-Nomad-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))

		return "", fmt.Errorf("openwonton node list failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var nodes []nodeRecord

	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return "", fmt.Errorf("failed to decode openwonton node list: %w", err)
	}

	trimmedNodeName := strings.TrimSpace(nodeName)

	for _, node := range nodes {
		if strings.TrimSpace(node.Name) == trimmedNodeName && strings.TrimSpace(node.ID) != "" {
			return node.ID, nil
		}
	}

	if len(nodes) == 1 && strings.TrimSpace(nodes[0].ID) != "" {
		return nodes[0].ID, nil
	}

	if trimmedNodeName == "" {
		return "", ErrNodeNotFound
	}

	return "", fmt.Errorf("%w: %q", ErrNodeNotFound, trimmedNodeName)
}

func setNodeEligibility(ctx context.Context, client *http.Client, baseURL, nodeID, eligibility, token string) error {
	payload, err := json.Marshal(map[string]string{
		"Eligibility": eligibility,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/v1/node/%s/eligibility", strings.TrimRight(baseURL, "/"), nodeID),
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("X-Nomad-Token", token)
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

	return fmt.Errorf("openwonton node eligibility update failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func enableNodeDrain(ctx context.Context, client *http.Client, baseURL, nodeID string, deadline time.Duration, token string) error {
	payload, err := json.Marshal(map[string]any{
		"DrainSpec": map[string]any{
			"Deadline":         deadline.Nanoseconds(),
			"IgnoreSystemJobs": false,
		},
		// Mark the node eligible again once draining completes so upgrades/reboots don't
		// permanently cordon it.
		"MarkEligible": true,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/v1/node/%s/drain", strings.TrimRight(baseURL, "/"), nodeID),
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("X-Nomad-Token", token)
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

	return fmt.Errorf("openwonton node drain failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}
