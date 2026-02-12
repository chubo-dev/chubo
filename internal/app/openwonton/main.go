// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package openwonton provides a minimal Nomad-compatible fallback API for chubo.
package openwonton

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type nomadRegisterRequest struct {
	Job struct {
		ID string `json:"ID"`
	} `json:"Job"`
}

// Main runs the OpenWonton fallback server.
//
// This is a temporary local fallback until real openwonton artifacts are present.
func Main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var (
		jobsMu sync.RWMutex
		jobs   = make(map[string][]byte)
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/status/leader", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `"127.0.0.1:4647"`)
	})

	mux.HandleFunc("/v1/job/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		jobID := strings.TrimPrefix(r.URL.Path, "/v1/job/")
		jobID = strings.TrimSpace(jobID)
		if jobID == "" {
			http.NotFound(w, r)

			return
		}

		jobsMu.RLock()
		payload, ok := jobs[jobID]
		jobsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"job not found"}`)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})

	mux.HandleFunc("/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"failed to read body"}`)

			return
		}

		var req nomadRegisterRequest
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid JSON payload"}`)

			return
		}

		jobID := strings.TrimSpace(req.Job.ID)
		if jobID == "" {
			jobID = "openbao"
		}

		jobsMu.Lock()
		jobs[jobID] = body
		jobsMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"EvalID":"mock-openwonton-eval"}`)
	})

	server := &http.Server{
		Addr:              ":4646",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("openwonton mock listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("openwonton mock failed: %v", err)
	}
}
