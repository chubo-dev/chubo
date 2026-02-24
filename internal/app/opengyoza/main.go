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
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	mockConsulDatacenter = "dc1"
	mockConsulVersion    = "1.16.4"
)

type agentServiceRegistration struct {
	ID      string            `json:"ID"`
	Name    string            `json:"Name"`
	Service string            `json:"Service"`
	Tags    []string          `json:"Tags"`
	Address string            `json:"Address"`
	Port    int               `json:"Port"`
	Meta    map[string]string `json:"Meta"`
}

type agentService struct {
	ID      string            `json:"ID"`
	Service string            `json:"Service"`
	Tags    []string          `json:"Tags,omitempty"`
	Address string            `json:"Address,omitempty"`
	Port    int               `json:"Port,omitempty"`
	Meta    map[string]string `json:"Meta,omitempty"`
}

func normalizeService(raw agentServiceRegistration) agentService {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = strings.TrimSpace(raw.Service)
	}

	id := strings.TrimSpace(raw.ID)
	if id == "" {
		id = name
	}

	addr := strings.TrimSpace(raw.Address)
	if addr == "" {
		addr = "127.0.0.1"
	}

	if raw.Meta == nil {
		raw.Meta = map[string]string{}
	}

	return agentService{
		ID:      id,
		Service: name,
		Tags:    raw.Tags,
		Address: addr,
		Port:    raw.Port,
		Meta:    raw.Meta,
	}
}

// Main runs the OpenGyoza fallback server.
//
// This is a temporary local fallback until real opengyoza artifacts are present.
func Main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mux := http.NewServeMux()
	var (
		servicesMu sync.RWMutex
		services   = map[string]agentService{}
	)

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
		writeJSON(w, []string{mockConsulDatacenter})
	})

	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"Config": map[string]any{
				"Datacenter": mockConsulDatacenter,
				"NodeName":   "opengyoza-mock",
				"Revision":   "mock",
				"Server":     true,
				"Version":    mockConsulVersion,
			},
		})
	})

	mux.HandleFunc("/v1/agent/members", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]any{
			{
				"Name": "opengyoza-mock",
				"Addr": "127.0.0.1",
				"Port": 8301,
			},
		})
	})

	mux.HandleFunc("/v1/agent/services", func(w http.ResponseWriter, _ *http.Request) {
		servicesMu.RLock()
		defer servicesMu.RUnlock()

		writeJSON(w, services)
	})

	mux.HandleFunc("/v1/agent/service/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		defer r.Body.Close()

		var req agentServiceRegistration
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		svc := normalizeService(req)
		if svc.ID == "" || svc.Service == "" {
			http.Error(w, "service id/name required", http.StatusBadRequest)

			return
		}

		servicesMu.Lock()
		services[svc.ID] = svc
		servicesMu.Unlock()

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1/agent/service/deregister/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v1/agent/service/deregister/")
		id = strings.TrimSpace(id)

		if id == "" {
			http.Error(w, "service id required", http.StatusBadRequest)

			return
		}

		servicesMu.Lock()
		delete(services, id)
		servicesMu.Unlock()

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1/agent/checks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{})
	})

	mux.HandleFunc("/v1/agent/check/register", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1/agent/check/deregister/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1/catalog/services", func(w http.ResponseWriter, _ *http.Request) {
		servicesMu.RLock()
		defer servicesMu.RUnlock()

		catalog := map[string][]string{}
		for _, svc := range services {
			catalog[svc.Service] = svc.Tags
		}

		writeJSON(w, catalog)
	})

	mux.HandleFunc("/v1/catalog/service/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/v1/catalog/service/")
		name = strings.TrimSpace(name)

		servicesMu.RLock()
		defer servicesMu.RUnlock()

		resp := make([]map[string]any, 0)
		for _, svc := range services {
			if name != "" && svc.Service != name {
				continue
			}

			resp = append(resp, map[string]any{
				"Node":           "opengyoza-mock",
				"Address":        "127.0.0.1",
				"Datacenter":     mockConsulDatacenter,
				"ServiceID":      svc.ID,
				"ServiceName":    svc.Service,
				"ServiceAddress": svc.Address,
				"ServicePort":    svc.Port,
				"ServiceTags":    svc.Tags,
			})
		}

		writeJSON(w, resp)
	})

	// Nomad and Traefik query health service endpoints for server discovery and catalog.
	mux.HandleFunc("/v1/health/service/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/v1/health/service/")
		name = strings.TrimSpace(name)

		servicesMu.RLock()
		defer servicesMu.RUnlock()

		resp := make([]map[string]any, 0)
		for _, svc := range services {
			if name != "" && svc.Service != name {
				continue
			}

			resp = append(resp, map[string]any{
				"Node": map[string]any{
					"Node":       "opengyoza-mock",
					"Address":    "127.0.0.1",
					"Datacenter": mockConsulDatacenter,
				},
				"Service": map[string]any{
					"ID":      svc.ID,
					"Service": svc.Service,
					"Address": svc.Address,
					"Port":    svc.Port,
					"Tags":    svc.Tags,
					"Meta":    svc.Meta,
				},
				"Checks": []map[string]any{
					{
						"Node":    "opengyoza-mock",
						"Name":    "service:" + svc.ID,
						"Status":  "passing",
						"Service": svc.Service,
					},
				},
			})
		}

		writeJSON(w, resp)
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
