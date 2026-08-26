package rest

import (
	"fmt"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// How long an anonymous read may be served from Cloudflare's edge.
//
// The same endpoint serves a live leaderboard and a History page fanning out across every
// cup, so the discriminator is not the route, it is the cup's phase. Handlers that can tell
// say so with cacheByPhase; everything else takes the default, which is short enough not to
// strand a correction and long enough to absorb a burst.
const (
	defaultMaxAge = 60
	settledMaxAge = 3600 // a finished cup, but a reset can still change one — see cacheByPhase
	noCacheMaxAge = 0
)

// cacheableRead marks anonymous successful reads as edge-cacheable, everything else no-store.
//
// Only scorers hold a token, and they refetch straight after submitting a hole, so serving
// them their own pre-submission data is the one staleness that would mislead. The header is
// set at WriteHeader to reflect the real status: a cached 404 would outlive its cause, and
// browsers honour Cache-Control on an error as readily as on a success.
func cacheableRead(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		next(&cacheWriter{
			ResponseWriter: w,
			anonymous:      req.Header.Get("Authorization") == "",
			maxAge:         defaultMaxAge,
		}, req)
	}
}

// cacheByPhase picks a tier from where the cup stands. A cup being played is exempt: it is
// read to find out what changed in the last minute, which is what a cache would withhold.
//
// A finished cup gets an hour rather than a day because resetting a match can unfinish one,
// and the endpoints publishing a cup disagree until the longest tier expires. An hour is what
// an operator can wait out; a day is not.
func cacheByPhase(w http.ResponseWriter, phase sdk.TournamentPhase) {
	switch phase {
	case sdk.PhaseLive:
		setMaxAge(w, noCacheMaxAge)
	case sdk.PhaseFinished:
		setMaxAge(w, settledMaxAge)
	}
}

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
