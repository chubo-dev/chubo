// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package openwontondrain

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testDrainDeadline = 10 * time.Minute

func TestReadRole(t *testing.T) {
	t.Parallel()

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		role, configured, err := ReadRole(filepath.Join(t.TempDir(), "missing"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if configured {
			t.Fatalf("expected configured=false")
		}

		if role != "" {
			t.Fatalf("expected empty role, got %q", role)
		}
	})

	t.Run("role file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "openwonton.role")
		if err := os.WriteFile(path, []byte(" client \n"), 0o644); err != nil {
			t.Fatalf("failed writing role file: %v", err)
		}

		role, configured, err := ReadRole(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !configured {
			t.Fatalf("expected configured=true")
		}

		if !IsClientRole(role) {
			t.Fatalf("expected client role, got %q", role)
		}
	})

	t.Run("hybrid role file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "openwonton.role")
		if err := os.WriteFile(path, []byte(" server-client \n"), 0o644); err != nil {
			t.Fatalf("failed writing role file: %v", err)
		}

		role, configured, err := ReadRole(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !configured {
			t.Fatalf("expected configured=true")
		}

		if !IsClientRole(role) {
			t.Fatalf("expected client-capable role, got %q", role)
		}
	})
}

func TestDrainNodeSuccess(t *testing.T) {
	t.Parallel()

	var sawEligibility, sawDrain bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/nodes":
			_ = json.NewEncoder(w).Encode([]nodeRecord{
				{ID: "node-1", Name: "client-a"},
				{ID: "node-2", Name: "client-b"},
			})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v1/node/node-1/eligibility":
			var payload map[string]string

			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed decoding eligibility payload: %v", err)
			}

			if payload["Eligibility"] != "ineligible" {
				t.Fatalf("unexpected eligibility payload: %#v", payload)
			}

			sawEligibility = true
			w.WriteHeader(http.StatusOK)
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v1/node/node-1/drain":
			var payload map[string]any

			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed decoding drain payload: %v", err)
			}

			drainSpec, ok := payload["DrainSpec"].(map[string]any)
			if !ok {
				t.Fatalf("drain payload missing DrainSpec: %#v", payload)
			}

			if int64(drainSpec["Deadline"].(float64)) != testDrainDeadline.Nanoseconds() {
				t.Fatalf("unexpected drain deadline: %#v", drainSpec["Deadline"])
			}

			if !payload["MarkEligible"].(bool) {
				t.Fatalf("expected MarkEligible=true")
			}

			sawDrain = true
			w.WriteHeader(http.StatusOK)
			return
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	if err := DrainNode(context.Background(), client, server.URL, "client-a", testDrainDeadline); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if !sawEligibility || !sawDrain {
		t.Fatalf("expected both eligibility and drain requests, got eligibility=%v drain=%v", sawEligibility, sawDrain)
	}
}

func TestDrainNodeSingleNodeFallback(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/nodes":
			_ = json.NewEncoder(w).Encode([]nodeRecord{
				{ID: "solo-node", Name: "some-other-name"},
			})
			return
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/node/solo-node/"):
			w.WriteHeader(http.StatusOK)
			return
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	if err := DrainNode(context.Background(), client, server.URL, "missing-name", testDrainDeadline); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestDrainNodeNoMatch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/nodes" {
			_ = json.NewEncoder(w).Encode([]nodeRecord{
				{ID: "node-1", Name: "client-a"},
				{ID: "node-2", Name: "client-b"},
			})

			return
		}

		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	err := DrainNode(context.Background(), client, server.URL, "missing", testDrainDeadline)
	if err == nil {
		t.Fatalf("expected error")
	}

	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}
