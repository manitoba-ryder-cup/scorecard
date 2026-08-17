package test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// The routing table in internal/api/rest/server.go is the only statement of which scope a
// write needs, and a wrong one there fails silently in both directions: the intended
// caller starts getting 403s, and a caller holding some other scope quietly gains the
// endpoint. Thirteen of the seventeen writes take scorecard:tournaments:write, so it is
// also what a copy-paste lands on.
//
// This table is deliberately a second, independent copy of that mapping. It is not
// derived from the server's registration — if it were, the two would agree by
// construction and prove nothing.
var scopedRoutes = []struct {
	method, route, scope string
}{
	{"POST", sdk.RouteV1Players, sdk.ScopePlayersWrite},
	{"PUT", sdk.RouteV1Player, sdk.ScopePlayersWrite},

	{"POST", sdk.RouteV1TeeColors, sdk.ScopeCoursesWrite},
	{"POST", sdk.RouteV1Courses, sdk.ScopeCoursesWrite},
	{"POST", sdk.RouteV1CourseTees, sdk.ScopeCoursesWrite},

	{"POST", sdk.RouteV1MatchScores, sdk.ScopeScoresWrite},

	{"POST", sdk.RouteV1Tournaments, sdk.ScopeTournamentsWrite},
	{"POST", sdk.RouteV1TournamentMatches, sdk.ScopeTournamentsWrite},
	{"PUT", sdk.RouteV1Match, sdk.ScopeTournamentsWrite},
	{"POST", sdk.RouteV1MatchParticipants, sdk.ScopeTournamentsWrite},
	{"DELETE", sdk.RouteV1MatchParticipant, sdk.ScopeTournamentsWrite},
	{"POST", sdk.RouteV1TournamentPlayers, sdk.ScopeTournamentsWrite},
	{"PUT", sdk.RouteV1TournamentPlayer, sdk.ScopeTournamentsWrite},
	{"POST", sdk.RouteV1TeamMembers, sdk.ScopeTournamentsWrite},
	{"DELETE", sdk.RouteV1TeamMember, sdk.ScopeTournamentsWrite},
	{"PUT", sdk.RouteV1TeamCaptain, sdk.ScopeTournamentsWrite},
	{"DELETE", sdk.RouteV1TeamCaptain, sdk.ScopeTournamentsWrite},
}

// concretePath fills the path parameters with UUIDs that match nothing. Authorization is
// checked before the handler looks anything up, so a route that rejects the request never
// reaches the point where the ID would matter — and one that accepts it answers 400 or
// 404, which is the proof we want.
func concretePath(route string) string {
	path := strings.ReplaceAll(route, "{id}", uuid.NewString())
	return strings.ReplaceAll(path, "{playerId}", uuid.NewString())
}

func TestEveryWriteRequiresItsOwnScope(t *testing.T) {
	t.Parallel()

	if len(scopedRoutes) != 17 {
		t.Fatalf("the table lists %d routes; server.go registers 17 scoped writes, so one "+
			"side has gained a route the other has not", len(scopedRoutes))
	}

	for _, r := range scopedRoutes {
		t.Run(r.method+" "+r.route, func(t *testing.T) {
			t.Parallel()
			path := concretePath(r.route)

			if status, body := request.Raw(t, r.method, path, "{}", ""); status != http.StatusUnauthorized {
				t.Fatalf("no token: want 401, got %d (%s)", status, body)
			}

			// Every other real scope must be refused. A token carrying one of these is an
			// ordinary caller with different privileges, not an attacker — which is exactly
			// the confusion a mis-registered route creates.
			for _, other := range writeScopes {
				if other == r.scope {
					continue
				}
				token := testjwt.MintAccessToken(t, uuid.New(), uuid.New(), other)
				if status, body := request.Raw(t, r.method, path, "{}", token); status != http.StatusForbidden {
					t.Fatalf("token holding only %s: want 403, got %d (%s)", other, status, body)
				}
			}

			// The declared scope must get past the middleware. What the handler then makes
			// of an empty body and unknown IDs is not this test's business — only that the
			// refusal is no longer about authorization.
			token := testjwt.MintAccessToken(t, uuid.New(), uuid.New(), r.scope)
			status, body := request.Raw(t, r.method, path, "{}", token)
			if status == http.StatusUnauthorized || status == http.StatusForbidden {
				t.Fatalf("token holding %s: still refused with %d (%s) — the route requires "+
					"a different scope than the table says", r.scope, status, body)
			}
		})
	}
}
