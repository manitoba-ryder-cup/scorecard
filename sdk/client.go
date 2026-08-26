package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client is an HTTP client for the scorecard API. It is the public contract's
// consumer half: callers (the web frontend, integration tests) speak the same DTOs
// the server emits, so drift between the two shows up at compile time.
type Client struct {
	http    *http.Client
	baseURL string
	token   string
}

// NewClient creates a scorecard API client targeting baseURL (e.g. http://localhost:5000).
func NewClient(baseURL string) *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// SetToken sets the bearer token sent on subsequent authenticated requests.
func (c *Client) SetToken(token string) { c.token = token }

// validatable is implemented by request bodies that can validate their own shape.
// The client runs it before sending, so bad input fails fast without a round trip.
type validatable interface {
	Validate(ctx context.Context) error
}

func (c *Client) ListPlayers(ctx context.Context) ([]PlayerProfile, error) {
	var out []PlayerProfile
	return out, c.do(ctx, http.MethodGet, RouteV1Players, nil, &out)
}

func (c *Client) GetPlayer(ctx context.Context, id uuid.UUID) (*PlayerProfile, error) {
	var out PlayerProfile
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1Player, id), nil, &out)
}

func (c *Client) CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*Player, error) {
	var out Player
	return &out, c.do(ctx, http.MethodPost, RouteV1Players, req, &out)
}

func (c *Client) UpdatePlayer(ctx context.Context, id uuid.UUID, req UpdatePlayerRequest) (*Player, error) {
	var out Player
	return &out, c.do(ctx, http.MethodPut, pathID(RouteV1Player, id), req, &out)
}

func (c *Client) GetPlayerTournaments(ctx context.Context, id uuid.UUID) ([]PlayerTournamentHistory, error) {
	var out []PlayerTournamentHistory
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1PlayerTournaments, id), nil, &out)
}

func (c *Client) GetPlayerStats(ctx context.Context, id uuid.UUID) (PlayerStats, error) {
	var out PlayerStats
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1PlayerStats, id), nil, &out)
}

func (c *Client) ListMatchFormats(ctx context.Context) ([]MatchFormat, error) {
	var out []MatchFormat
	return out, c.do(ctx, http.MethodGet, RouteV1MatchFormats, nil, &out)
}

func (c *Client) ListTeeColors(ctx context.Context) ([]TeeColor, error) {
	var out []TeeColor
	return out, c.do(ctx, http.MethodGet, RouteV1TeeColors, nil, &out)
}

func (c *Client) CreateTeeColor(ctx context.Context, req CreateTeeColorRequest) (*TeeColor, error) {
	var out TeeColor
	return &out, c.do(ctx, http.MethodPost, RouteV1TeeColors, req, &out)
}

func (c *Client) ListCourses(ctx context.Context) ([]Course, error) {
	var out []Course
	return out, c.do(ctx, http.MethodGet, RouteV1Courses, nil, &out)
}

func (c *Client) GetCourse(ctx context.Context, id uuid.UUID) (*Course, error) {
	var out Course
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1Course, id), nil, &out)
}

func (c *Client) CreateCourse(ctx context.Context, req CreateCourseRequest) (*Course, error) {
	var out Course
	return &out, c.do(ctx, http.MethodPost, RouteV1Courses, req, &out)
}

// AddTeeSet adds a tee set and its 18 holes to a course.
func (c *Client) AddTeeSet(ctx context.Context, courseID uuid.UUID, req CreateTeeSetRequest) (*TeeSet, error) {
	var out TeeSet
	return &out, c.do(ctx, http.MethodPost, pathID(RouteV1CourseTees, courseID), req, &out)
}

// ListCourseTeeSets lists a course's configured tee sets (with colour names) — the valid
// (course, tee) options for setting up a match.
func (c *Client) ListCourseTeeSets(ctx context.Context, courseID uuid.UUID) ([]TeeSetSummary, error) {
	var out []TeeSetSummary
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1CourseTees, courseID), nil, &out)
}

func (c *Client) ListTournaments(ctx context.Context) ([]Tournament, error) {
	var out []Tournament
	return out, c.do(ctx, http.MethodGet, RouteV1Tournaments, nil, &out)
}

func (c *Client) GetTournament(ctx context.Context, id uuid.UUID) (*Tournament, error) {
	var out Tournament
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1Tournament, id), nil, &out)
}

// CreateTournament creates a tournament; the server also seeds its two teams.
func (c *Client) CreateTournament(ctx context.Context, req CreateTournamentRequest) (*Tournament, error) {
	var out Tournament
	return &out, c.do(ctx, http.MethodPost, RouteV1Tournaments, req, &out)
}

func (c *Client) GetTournamentTeams(ctx context.Context, id uuid.UUID) ([]TournamentTeam, error) {
	var out []TournamentTeam
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentTeams, id), nil, &out)
}

func (c *Client) GetTournamentWinner(ctx context.Context, id uuid.UUID) (*WinnerResponse, error) {
	var out WinnerResponse
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentWinner, id), nil, &out)
}

func (c *Client) GetTournamentResults(ctx context.Context, id uuid.UUID) ([]MatchResult, error) {
	var out []MatchResult
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentResults, id), nil, &out)
}

func (c *Client) GetTournamentStatus(ctx context.Context, id uuid.UUID) (*FinishedResponse, error) {
	var out FinishedResponse
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentStatus, id), nil, &out)
}

func (c *Client) ListTournamentPlayers(ctx context.Context, tournamentID uuid.UUID) ([]TournamentPlayer, error) {
	var out []TournamentPlayer
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentPlayers, tournamentID), nil, &out)
}

func (c *Client) EnterTournamentPlayer(ctx context.Context, tournamentID uuid.UUID, req EnterTournamentPlayerRequest) (*TournamentPlayer, error) {
	var out TournamentPlayer
	return &out, c.do(ctx, http.MethodPost, pathID(RouteV1TournamentPlayers, tournamentID), req, &out)
}

func (c *Client) UpdateTournamentPlayer(ctx context.Context, tournamentID, playerID uuid.UUID, req UpdateTournamentPlayerRequest) (*TournamentPlayer, error) {
	route := strings.Replace(pathID(RouteV1TournamentPlayer, tournamentID), "{playerId}", playerID.String(), 1)
	var out TournamentPlayer
	return &out, c.do(ctx, http.MethodPut, route, req, &out)
}

// DraftPlayer assigns an entered player to a team.
func (c *Client) DraftPlayer(ctx context.Context, teamID uuid.UUID, req DraftPlayerRequest) (*TeamMember, error) {
	var out TeamMember
	return &out, c.do(ctx, http.MethodPost, pathID(RouteV1TeamMembers, teamID), req, &out)
}

// SetTeamCaptain assigns a team's captain. A 204 (no body) is success.
func (c *Client) SetTeamCaptain(ctx context.Context, teamID uuid.UUID, req SetTeamCaptainRequest) error {
	return c.do(ctx, http.MethodPut, pathID(RouteV1TeamCaptain, teamID), req, nil)
}

// ClearTeamCaptain leaves a team with no captain. 204 is success; 404 if the team doesn't
// exist. Naming a different captain is SetTeamCaptain on its own.
func (c *Client) ClearTeamCaptain(ctx context.Context, teamID uuid.UUID) error {
	return c.do(ctx, http.MethodDelete, pathID(RouteV1TeamCaptain, teamID), nil, nil)
}

// UndraftPlayer removes a player from a team. A 204 (no body) is success; 404 if they
// weren't on the team, and 409 while they are named in a match lineup.
func (c *Client) UndraftPlayer(ctx context.Context, teamID, playerID uuid.UUID) error {
	route := strings.Replace(pathID(RouteV1TeamMember, teamID), "{playerId}", playerID.String(), 1)
	return c.do(ctx, http.MethodDelete, route, nil, nil)
}

// ListTeamMembers lists a team's drafted players (the roster-entry view, filtered).
func (c *Client) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]TournamentPlayer, error) {
	var out []TournamentPlayer
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1TeamMembers, teamID), nil, &out)
}

func (c *Client) ListMatches(ctx context.Context, tournamentID uuid.UUID) ([]Match, error) {
	var out []Match
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentMatches, tournamentID), nil, &out)
}

func (c *Client) CreateMatch(ctx context.Context, tournamentID uuid.UUID, req CreateMatchRequest) (*Match, error) {
	var out Match
	return &out, c.do(ctx, http.MethodPost, pathID(RouteV1TournamentMatches, tournamentID), req, &out)
}

func (c *Client) UpdateMatch(ctx context.Context, matchID uuid.UUID, req UpdateMatchRequest) (*Match, error) {
	var out Match
	return &out, c.do(ctx, http.MethodPut, pathID(RouteV1Match, matchID), req, &out)
}

func (c *Client) DeleteMatch(ctx context.Context, matchID uuid.UUID) error {
	return c.do(ctx, http.MethodDelete, pathID(RouteV1Match, matchID), nil, nil)
}

func (c *Client) ListParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error) {
	var out []MatchParticipant
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchParticipants, matchID), nil, &out)
}

// SetLineup replaces a match's lineup with the one given. A 204 (no body) is success.
func (c *Client) SetLineup(ctx context.Context, matchID uuid.UUID, req SetLineupRequest) error {
	return c.do(ctx, http.MethodPut, pathID(RouteV1MatchParticipants, matchID), req, nil)
}

// SubmitScore records a hole's scores and returns the match's recomputed state, so a
// caller knows whether that hole closed the match out. The hole lands whole or not at all.
func (c *Client) SubmitScore(ctx context.Context, matchID uuid.UUID, req ScoreSubmission) (MatchStatus, error) {
	var out MatchStatus
	return out, c.do(ctx, http.MethodPost, pathID(RouteV1MatchScores, matchID), req, &out)
}

func (c *Client) ResetMatchScores(ctx context.Context, matchID uuid.UUID) error {
	return c.do(ctx, http.MethodDelete, pathID(RouteV1MatchScores, matchID), nil, nil)
}

func (c *Client) GetMatchScores(ctx context.Context, matchID uuid.UUID) ([]HoleStatus, error) {
	var out []HoleStatus
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchScores, matchID), nil, &out)
}

func (c *Client) GetMatchHoles(ctx context.Context, matchID uuid.UUID) ([]Hole, error) {
	var out []Hole
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchHoles, matchID), nil, &out)
}

// GetMatchStatus reads a match's outcome. RouteV1MatchWinner answers identically.
func (c *Client) GetMatchStatus(ctx context.Context, matchID uuid.UUID) (*MatchStatus, error) {
	var out MatchStatus
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchStatus, matchID), nil, &out)
}

// pathID substitutes the {id} path parameter in a route template.
func pathID(route string, id uuid.UUID) string {
	return strings.Replace(route, "{id}", id.String(), 1)
}

// do performs a request, attaching the bearer token, encoding req (if any) as JSON,
// and decoding a success body into result (if non-nil). Non-2xx responses become
// *APIError carrying the server's error message.
func (c *Client) do(ctx context.Context, method, route string, req, result any) error {
	var body io.Reader
	if req != nil {
		// Client-side fast-fail: reject a malformed request before it hits the wire.
		if v, ok := req.(validatable); ok {
			if err := v.Validate(ctx); err != nil {
				return fmt.Errorf("invalid request: %w", err)
			}
		}
		b, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+route, body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if req != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return decodeAPIError(resp)
	}
	if result == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// decodeAPIError builds an *APIError from a non-2xx response, preferring the SDK's
// {"error": "..."} envelope and falling back to the raw body.
func decodeAPIError(resp *http.Response) error {
	respBody, _ := io.ReadAll(resp.Body)
	var errResp ErrorResponse
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
		return &APIError{StatusCode: resp.StatusCode, Message: errResp.Error}
	}
	return &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
}
