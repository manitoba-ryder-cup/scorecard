// Package request sends HTTP requests straight to the server, bypassing the SDK
// client's validation. It exists so negative tests can prove the server validates
// too — a non-SDK caller sending bad input must still be rejected.
package request

import (
	"io"
	"net/http"
	"strings"
	"testing"

	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// Raw sends method+path with the given JSON body and bearer token, returning the
// response status code and body.
func Raw(t *testing.T, method, path, body, accessToken string) (int, string) {
	t.Helper()

	cfg := util.LoadConfig()
	req, err := http.NewRequest(method, cfg.BaseURL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, string(respBody)
}
