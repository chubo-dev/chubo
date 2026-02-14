package opengyozaleave

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLeaveWithToken(t *testing.T) {
	t.Parallel()

	const token = "tok"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%q want=%q", r.Method, http.MethodPut)
		}

		if r.URL.Path != "/v1/agent/leave" {
			t.Fatalf("path=%q want=%q", r.URL.Path, "/v1/agent/leave")
		}

		if got := r.Header.Get("X-Consul-Token"); got != token {
			t.Fatalf("X-Consul-Token=%q want=%q", got, token)
		}

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)

	if err := LeaveWithToken(ctx, srv.Client(), srv.URL, token); err != nil {
		t.Fatalf("LeaveWithToken() err=%v", err)
	}
}
