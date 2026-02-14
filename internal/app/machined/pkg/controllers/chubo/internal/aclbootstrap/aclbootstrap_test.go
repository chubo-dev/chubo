package aclbootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBootstrapACL_AlreadyBootstrapped_StatusConflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("already bootstrapped"))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	already, err := bootstrapACL(ctx, srv.Client(), http.MethodPut, srv.URL+"/v1/acl/bootstrap", "token")
	if err != nil {
		t.Fatalf("bootstrapACL error: %v", err)
	}

	if !already {
		t.Fatalf("expected alreadyBootstrapped=true")
	}
}

func TestEnsureConsulACL_AlreadyBootstrapped_TokenMismatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/acl/token/self":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("permission denied"))
		case "/v1/acl/bootstrap":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("ACL bootstrap already done"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	ok, err := EnsureConsulACL(ctx, srv.Client(), srv.URL, "derived-token", true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if ok {
		t.Fatalf("expected ok=false")
	}

	msg := err.Error()
	if !strings.Contains(msg, "already bootstrapped") || !strings.Contains(msg, "trust.token") {
		t.Fatalf("expected mismatch error mentioning already-bootstrapped + trust.token, got: %q", msg)
	}
}

func TestEnsureNomadACL_AlreadyBootstrapped_TokenMismatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/acl/token/self":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("permission denied"))
		case "/v1/acl/bootstrap":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":["ACL bootstrap already done"]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	ok, err := EnsureNomadACL(ctx, srv.Client(), srv.URL, "derived-token", true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if ok {
		t.Fatalf("expected ok=false")
	}

	msg := err.Error()
	if !strings.Contains(msg, "already bootstrapped") || !strings.Contains(msg, "trust.token") {
		t.Fatalf("expected mismatch error mentioning already-bootstrapped + trust.token, got: %q", msg)
	}
}
