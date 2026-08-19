package rest

import (
	"fmt"
	"net/http"
)

// How long an anonymous read may be served from Cloudflare's edge.
//
// The split exists because two very different requests hit the same endpoints. The
// History page fans out across every cup, and seventeen of eighteen can never change
// again. The live leaderboard polls /teams and /results every twenty seconds so a
// spectator sees scores land without refreshing — cache those and the polling is
// pointless, because two out of three requests would return the same stored answer.
//
// So the discriminator is not the route, it is whether the cup is over. Handlers that can
// tell say so with cacheSettled or cacheLive; everything else takes the default, which is
// short enough not to strand a correction and long enough to absorb a burst.
const (
	defaultMaxAge = 60
	settledMaxAge = 86400 // a finished cup: a day, so a late correction still surfaces
	noCacheMaxAge = 0
)

// cacheableRead marks anonymous successful reads as edge-cacheable, and everything else
// as no-store.
//
// Authenticated requests are never cached. Only scorers hold a token: they submit a hole
// and immediately refetch, and serving them their own pre-submission data would be the one
// genuinely misleading staleness. Spectators carry no token and cannot write.
//
// The header is set at WriteHeader so it reflects the real status: a cached 404 or 500
// would outlive the condition that caused it, and browsers honour Cache-Control on error
// responses just as readily as on successful ones.
func cacheableRead(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		next(&cacheWriter{
			ResponseWriter: w,
			anonymous:      req.Header.Get("Authorization") == "",
			maxAge:         defaultMaxAge,
		}, req)
	}
}

// cacheSettled marks a response that describes a finished cup — nothing about it can
// change again, so it is worth caching properly rather than for a minute.
func cacheSettled(w http.ResponseWriter) { setMaxAge(w, settledMaxAge) }

// cacheLive marks a response a spectator is polling for. Caching it at all would defeat
// the poll, so it is served fresh every time.
func cacheLive(w http.ResponseWriter) { setMaxAge(w, noCacheMaxAge) }

// setMaxAge is a no-op when the route was not registered as a cacheable read, so a
// handler can call it unconditionally.
func setMaxAge(w http.ResponseWriter, seconds int) {
	if c, ok := w.(*cacheWriter); ok {
		c.maxAge = seconds
	}
}

type cacheWriter struct {
	http.ResponseWriter
	anonymous bool
	maxAge    int
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
	if c.anonymous && c.maxAge > 0 && status >= 200 && status < 300 {
		c.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", c.maxAge))
		return
	}
	c.Header().Set("Cache-Control", "no-store")
}
