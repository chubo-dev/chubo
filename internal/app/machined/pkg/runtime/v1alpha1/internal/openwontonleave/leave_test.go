package openwontonleave

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPurgeNodeWithToken(t *testing.T) {
	t.Parallel()

	const (
		token    = "tok"
		nodeName = "node-a"
		nodeID   = "node-1"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/nodes":
			if r.Method != http.MethodGet {
				t.Fatalf("nodes method=%q want=%q", r.Method, http.MethodGet)
			}
			if got := r.Header.Get(nomadTokenHeader); got != token {
				t.Fatalf("%s=%q want=%q", nomadTokenHeader, got, token)
			}

			_ = json.NewEncoder(w).Encode([]nodeRecord{{ID: nodeID, Name: nodeName}})
		case "/v1/node/" + nodeID + "/purge":
			if r.Method != http.MethodPost {
				t.Fatalf("purge method=%q want=%q", r.Method, http.MethodPost)
			}
			if got := r.Header.Get(nomadTokenHeader); got != token {
				t.Fatalf("%s=%q want=%q", nomadTokenHeader, got, token)
			}

			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)

	if err := PurgeNodeWithToken(ctx, srv.Client(), srv.URL, nodeName, token); err != nil {
		t.Fatalf("PurgeNodeWithToken() err=%v", err)
	}
}

func TestRemoveServerPeerWithToken(t *testing.T) {
	t.Parallel()

	const (
		token    = "tok"
		nodeName = "node-a"
		peerID   = "peer-1"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(nomadTokenHeader); got != token {
			t.Fatalf("%s=%q want=%q", nomadTokenHeader, got, token)
		}

		switch r.URL.Path {
		case "/v1/operator/raft/configuration":
			if r.Method != http.MethodGet {
				t.Fatalf("configuration method=%q want=%q", r.Method, http.MethodGet)
			}

			_ = json.NewEncoder(w).Encode(raftConfiguration{
				Servers: []raftServer{{ID: peerID, Node: nodeName, Address: "10.0.0.1:4647", Voter: true}},
			})
		case "/v1/operator/raft/peer":
			if r.Method != http.MethodDelete {
				t.Fatalf("peer method=%q want=%q", r.Method, http.MethodDelete)
			}

			if got := strings.TrimSpace(r.URL.Query().Get("id")); got != peerID {
				t.Fatalf("query id=%q want=%q", got, peerID)
			}

			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)

	if err := RemoveServerPeerWithToken(ctx, srv.Client(), srv.URL, nodeName, token); err != nil {
		t.Fatalf("RemoveServerPeerWithToken() err=%v", err)
	}
}
