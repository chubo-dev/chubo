package aclbootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	nomadTokenHeader  = "X-Nomad-Token"
	consulTokenHeader = "X-Consul-Token"

	requestBodyLimit = 4096
)

var errMissingACLToken = errors.New("missing derived ACL token")

type bootstrapSecretRequest struct {
	BootstrapSecret string `json:"BootstrapSecret"`
}

type httpStatusError struct {
	status int
	body   string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("token self failed: status=%d body=%s", e.status, strings.TrimSpace(e.body))
}

func (e *httpStatusError) StatusCode() int {
	return e.status
}

func EnsureNomadACL(ctx context.Context, client *http.Client, baseURL, token string, allowBootstrap bool) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, errMissingACLToken
	}

	ok, selfErr := tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", nomadTokenHeader, token)
	if ok {
		return true, nil
	}

	if !allowBootstrap {
		return false, selfErr
	}

	alreadyBootstrapped, err := bootstrapACL(ctx, client, http.MethodPost, baseURL+"/v1/acl/bootstrap", token)
	if err != nil {
		return false, err
	}

	ok, err = tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", nomadTokenHeader, token)
	if ok {
		return true, nil
	}

	if err != nil {
		if alreadyBootstrapped && isTokenUnauthorized(err) {
			return false, fmt.Errorf("nomad ACL already bootstrapped but derived token is not accepted (likely trust.token rotated): %w", err)
		}

		return false, err
	}

	return false, fmt.Errorf("nomad ACL token not accepted after bootstrap")
}

func EnsureConsulACL(ctx context.Context, client *http.Client, baseURL, token string, allowBootstrap bool) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, errMissingACLToken
	}

	ok, selfErr := tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", consulTokenHeader, token)
	if ok {
		return true, nil
	}

	if !allowBootstrap {
		return false, selfErr
	}

	alreadyBootstrapped, err := bootstrapACL(ctx, client, http.MethodPut, baseURL+"/v1/acl/bootstrap", token)
	if err != nil {
		return false, err
	}

	ok, err = tokenSelf(ctx, client, baseURL+"/v1/acl/token/self", consulTokenHeader, token)
	if ok {
		return true, nil
	}

	if err != nil {
		if alreadyBootstrapped && isTokenUnauthorized(err) {
			sha := sha256.Sum256([]byte(token))
			// Include a short prefix to correlate with the COSI status field without leaking the token.
			return false, fmt.Errorf("consul ACL already bootstrapped but derived token is not accepted (likely trust.token rotated, tokenSHA=%x): %w", sha[:6], err)
		}

		return false, err
	}

	return false, fmt.Errorf("consul ACL token not accepted after bootstrap")
}

func tokenSelf(ctx context.Context, client *http.Client, url, header, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set(header, token)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))

	return false, &httpStatusError{status: resp.StatusCode, body: string(body)}
}

func isTokenUnauthorized(err error) bool {
	var statusErr *httpStatusError
	if !errors.As(err, &statusErr) {
		return false
	}

	return statusErr.StatusCode() == http.StatusUnauthorized || statusErr.StatusCode() == http.StatusForbidden
}

func bootstrapACL(ctx context.Context, client *http.Client, method, url, token string) (bool, error) {
	payload, err := json.Marshal(bootstrapSecretRequest{BootstrapSecret: token})
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, requestBodyLimit))
	bodyStr := strings.TrimSpace(string(body))
	bodyLower := strings.ToLower(bodyStr)

	// Treat "already bootstrapped" (or similar) as success to make the operation idempotent.
	if resp.StatusCode == http.StatusConflict || (strings.Contains(bodyLower, "bootstrap") && strings.Contains(bodyLower, "already")) {
		return true, nil
	}

	return false, fmt.Errorf("ACL bootstrap failed: status=%d body=%s", resp.StatusCode, bodyStr)
}
