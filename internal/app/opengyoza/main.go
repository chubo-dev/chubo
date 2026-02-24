// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package opengyoza provides a minimal Consul-compatible fallback API for chubo.
package opengyoza

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Main runs the OpenGyoza fallback server.
//
// This is a temporary local fallback until real opengyoza artifacts are present.
func Main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mux := http.NewServeMux()

	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	mux.HandleFunc("/v1/status/leader", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, "127.0.0.1:8300")
	})

	mux.HandleFunc("/v1/catalog/datacenters", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []string{"dc1"})
	})

	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"Config": map[string]any{
				"Datacenter": "dc1",
				"NodeName":   "opengyoza-mock",
				"Revision":   "mock",
				"Server":     true,
				"Version":    "1.16.4",
			},
		})
	})

	mux.HandleFunc("/v1/agent/members", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []any{})
	})

	// Nomad may query health service endpoints for server discovery.
	mux.HandleFunc("/v1/health/service/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []any{})
	})

	server := &http.Server{
		Addr:              ":8500",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("opengyoza mock listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("opengyoza mock failed: %v", err)
	}
}
