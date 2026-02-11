// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package main provides a minimal Chubo module workload process.
//
// The process is intentionally small: it validates that bootstrap data is
// readable and publishes a heartbeat file under /var/lib/chubo/state.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	bootstrapPath = "/var/lib/chubo/bootstrap/bootstrap.json"
	stateDir      = "/var/lib/chubo/state"
	statePath     = "/var/lib/chubo/state/chubo-agent-status.json"
	tickInterval  = 10 * time.Second
)

type agentStatus struct {
	UpdatedAtUTC      string `json:"updatedAtUTC"`
	BootstrapPresent  bool   `json:"bootstrapPresent"`
	BootstrapSHA256   string `json:"bootstrapSHA256,omitempty"`
	BootstrapByteSize int    `json:"bootstrapByteSize,omitempty"`
	LastError         string `json:"lastError,omitempty"`
}

func main() {
	log.Printf("chubo-agent: starting")

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		status := agentStatus{
			UpdatedAtUTC: time.Now().UTC().Format(time.RFC3339),
		}

		if err := reconcile(&status); err != nil {
			status.LastError = err.Error()
			log.Printf("chubo-agent: reconcile error: %v", err)
		}

		if err := writeStatus(statePath, status); err != nil {
			log.Printf("chubo-agent: status write error: %v", err)
		}

		<-ticker.C
	}
}

func reconcile(status *agentStatus) error {
	payload, err := os.ReadFile(bootstrapPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	sum := sha256.Sum256(payload)
	status.BootstrapPresent = true
	status.BootstrapSHA256 = hex.EncodeToString(sum[:])
	status.BootstrapByteSize = len(payload)

	return nil
}

func writeStatus(path string, status agentStatus) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return err
	}

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	tmpPath := filepath.Join(stateDir, ".chubo-agent-status.json.tmp")

	if err = os.WriteFile(tmpPath, append(data, '\n'), 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
