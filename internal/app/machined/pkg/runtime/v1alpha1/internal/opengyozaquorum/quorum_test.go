// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package opengyozaquorum

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

		path := filepath.Join(t.TempDir(), "opengyoza.role")
		if err := os.WriteFile(path, []byte(" server \n"), 0o644); err != nil {
			t.Fatalf("failed writing role file: %v", err)
		}

		role, configured, err := ReadRole(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !configured {
			t.Fatalf("expected configured=true")
		}

		if !IsServerRole(role) {
			t.Fatalf("expected server role, got %q", role)
		}
	})
}

func TestCheckSafeServerStop(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		peers       []string
		expectError bool
	}{
		{name: "single server is allowed", peers: []string{"127.0.0.1:8300"}, expectError: false},
		{name: "three servers is safe", peers: []string{"a", "b", "c"}, expectError: false},
		{name: "two servers is unsafe", peers: []string{"a", "b"}, expectError: true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/status/peers" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}

				_ = json.NewEncoder(w).Encode(tc.peers)
			}))
			defer server.Close()

			client := &http.Client{Timeout: time.Second}

			err := CheckSafeServerStop(context.Background(), client, server.URL)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}

				if !errors.Is(err, ErrUnsafeServerStop) {
					t.Fatalf("expected ErrUnsafeServerStop, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckSafeServerStopFromPeers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		peers       []string
		expectError bool
	}{
		{name: "single server is allowed", peers: []string{"127.0.0.1:8300"}, expectError: false},
		{name: "three servers is safe", peers: []string{"a", "b", "c"}, expectError: false},
		{name: "two servers is unsafe", peers: []string{"a", "b"}, expectError: true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := CheckSafeServerStopFromPeers(tc.peers)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}

				if !errors.Is(err, ErrUnsafeServerStop) {
					t.Fatalf("expected ErrUnsafeServerStop, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
