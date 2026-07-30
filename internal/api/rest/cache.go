package rest

import (
	"fmt"
	"net/http"
)

// publicReadMaxAge is how long an anonymous read may be served from Cloudflare's edge.
//
// Sixty seconds is chosen against how this app is actually used. Everything public is a
// spectator view, and the only one anyone watches closely is a live leaderboard — where a
// minute behind is invisible, because a fourball takes four hours and holes are entered
// one at a time. Against that, the History page alone fans out to eighteen requests, one
// per cup, and every one of them is immutable. Serving those from the edge skips both the
// Cloud Run cold start and the round trips to a database in another region.
const publicReadMaxAge = 60

// cacheableRead marks anonymous successful reads as edge-cacheable, and everything else
// as no-store.
//
// Authenticated requests are never cached, which is what keeps a live tournament correct.
// Only scorers hold a token: they submit a hole and immediately refetch, and serving them
// their own pre-submission data would be the one genuinely confusing staleness. Spectators
// carry no token and cannot write, so a minute-old view can never contradict something
// they just did.
//
// The header is set at WriteHeader so it reflects the real status: a cached 404 or 500
// would outlive the condition that caused it, and browsers honour Cache-Control on error
// responses just as readily as on successful ones.
func cacheableRead(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(&cacheWriter{ResponseWriter: w, anonymous: r.Header.Get("Authorization") == ""}, r)
	}
}

type cacheWriter struct {
	http.ResponseWriter
	anonymous bool
	decided   bool
}

func (c *cacheWriter) WriteHeader(status int) {
	c.decide(status)
	c.ResponseWriter.WriteHeader(status)
}

// Write covers a handler that never calls WriteHeader, which implies 200.
func (c *cacheWriter) Write(b []byte) (int, error) {
	c.decide(http.StatusOK)
	return c.ResponseWriter.Write(b)
}

func (c *cacheWriter) decide(status int) {
	if c.decided {
		return
	}
	c.decided = true
	if c.anonymous && status >= 200 && status < 300 {
		c.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", publicReadMaxAge))
		return
	}
	c.Header().Set("Cache-Control", "no-store")
}
