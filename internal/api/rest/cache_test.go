package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// handler that responds with the given status, the way respondJSON does.
func respondWith(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{}`))
	}
}

func cacheControlFor(t *testing.T, h http.HandlerFunc, authorized bool) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/v1/tournaments", nil)
	if authorized {
		req.Header.Set("Authorization", "Bearer token")
	}
	rec := httptest.NewRecorder()
	cacheableRead(h)(rec, req)
	return rec.Header().Get("Cache-Control")
}

func TestCacheableRead_AnonymousSuccessIsCacheable(t *testing.T) {
	if got := cacheControlFor(t, respondWith(http.StatusOK), false); got != "public, max-age=60" {
		t.Errorf("anonymous 200: got %q, want a public max-age", got)
	}
}

// The rule the live leaderboard depends on: a scorer holds a token, submits a hole, and
// immediately refetches. Serving them their own pre-submission data is the one staleness
// that would actually mislead someone.
func TestCacheableRead_AuthenticatedIsNeverCached(t *testing.T) {
	if got := cacheControlFor(t, respondWith(http.StatusOK), true); got != "no-store" {
		t.Errorf("authenticated 200: got %q, want no-store", got)
	}
}

// A cached error outlives whatever caused it, and browsers honour Cache-Control on error
// responses just as readily as on successful ones.
func TestCacheableRead_ErrorsAreNeverCached(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		if got := cacheControlFor(t, respondWith(status), false); got != "no-store" {
			t.Errorf("anonymous %d: got %q, want no-store", status, got)
		}
	}
}

// A handler that writes a body without calling WriteHeader has implicitly sent a 200, and
// must still come out cacheable.
func TestCacheableRead_ImplicitOK(t *testing.T) {
	implicit := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if got := cacheControlFor(t, implicit, false); got != "public, max-age=60" {
		t.Errorf("implicit 200: got %q, want a public max-age", got)
	}
}

// The status a handler sends must survive the wrapper untouched.
func TestCacheableRead_PassesStatusThrough(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tournaments", nil)
	rec := httptest.NewRecorder()
	cacheableRead(respondWith(http.StatusTeapot))(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusTeapot)
	}
}
