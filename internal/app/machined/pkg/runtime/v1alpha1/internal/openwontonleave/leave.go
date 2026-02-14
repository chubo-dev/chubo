// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package openwontonleave

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	roleClient       = "client"
	roleServer       = "server"
	requestBodyLimit = 4096

	nomadTokenHeader = "X-Nomad-Token"
)

var ErrNodeNotFound = errors.New("openwonton node not found")

type nodeRecord struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type raftConfiguration struct {
	Servers []raftServer `json:"Servers"`
}

type raftServer struct {
	ID      string `json:"ID"`
	Node    string `json:"Node"`
	Address string `json:"Address"`
	Leader  bool   `json:"Leader"`
	Voter   bool   `json:"Voter"`
}

func IsClientRole(role string) bool {
	return strings.TrimSpace(role) == roleClient
}

func IsServerRole(role string) bool {
	return strings.TrimSpace(role) == roleServer
}

func PurgeNode(ctx context.Context, client *http.Client, baseURL, nodeName string) error {
	return PurgeNodeWithToken(ctx, client, baseURL, nodeName, "")
}

func PurgeNodeWithToken(ctx context.Context, client *http.Client, baseURL, nodeName, token string) error {
	nodeID, err := resolveNodeID(ctx, client, baseURL, nodeName, token)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/v1/node/%s/purge", strings.TrimRight(baseURL, "/"), nodeID),
		nil,
	)
	if err != nil {
		return err
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set(nomadTokenHeader, token)
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

	return fmt.Errorf("openwonton node purge failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func RemoveServerPeer(ctx context.Context, client *http.Client, baseURL, nodeName string) error {
	return RemoveServerPeerWithToken(ctx, client, baseURL, nodeName, "")
}

func RemoveServerPeerWithToken(ctx context.Context, client *http.Client, baseURL, nodeName, token string) error {
	peerID, err := resolvePeerID(ctx, client, baseURL, nodeName, token)
	if err != nil {
		return err
	}

	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/v1/operator/raft/peer")
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("id", peerID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set(nomadTokenHeader, token)
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

	return fmt.Errorf("openwonton raft peer removal failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func resolveNodeID(ctx context.Context, client *http.Client, baseURL, nodeName, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/nodes", nil)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set(nomadTokenHeader, token)
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

func resolvePeerID(ctx context.Context, client *http.Client, baseURL, nodeName, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/operator/raft/configuration", nil)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set(nomadTokenHeader, token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("openwonton raft config failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var cfg raftConfiguration
	if err := json.Unmarshal(body, &cfg); err != nil {
		return "", fmt.Errorf("decode raft config: %w", err)
	}

	trimmedNode := strings.TrimSpace(nodeName)
	for _, server := range cfg.Servers {
		if strings.TrimSpace(server.ID) == "" {
			continue
		}

		if trimmedNode != "" && strings.TrimSpace(server.Node) == trimmedNode {
			return strings.TrimSpace(server.ID), nil
		}
	}

	// Best-effort fallback for single-node server clusters where Node might not match.
	if len(cfg.Servers) == 1 && strings.TrimSpace(cfg.Servers[0].ID) != "" {
		return strings.TrimSpace(cfg.Servers[0].ID), nil
	}

	if strings.TrimSpace(trimmedNode) == "" {
		return "", ErrNodeNotFound
	}

	return "", fmt.Errorf("%w: %q", ErrNodeNotFound, trimmedNode)
}
