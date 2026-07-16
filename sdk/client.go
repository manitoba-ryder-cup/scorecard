package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// --- Players ---

func (c *Client) ListPlayers(ctx context.Context) ([]Player, error) {
	var out []Player
	return out, c.do(ctx, http.MethodGet, RouteV1Players, nil, &out)
}

func (c *Client) GetPlayer(ctx context.Context, id int32) (*PlayerProfile, error) {
	var out PlayerProfile
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1Player, id), nil, &out)
}

func (c *Client) CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*Player, error) {
	var out Player
	return &out, c.do(ctx, http.MethodPost, RouteV1Players, req, &out)
}

// --- Tournaments ---

func (c *Client) ListTournaments(ctx context.Context) ([]Tournament, error) {
	var out []Tournament
	return out, c.do(ctx, http.MethodGet, RouteV1Tournaments, nil, &out)
}

func (c *Client) GetTournament(ctx context.Context, id int32) (*Tournament, error) {
	var out Tournament
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1Tournament, id), nil, &out)
}

// CreateTournament creates a tournament; the server also seeds its two teams.
func (c *Client) CreateTournament(ctx context.Context, req CreateTournamentRequest) (*Tournament, error) {
	var out Tournament
	return &out, c.do(ctx, http.MethodPost, RouteV1Tournaments, req, &out)
}

func (c *Client) GetTournamentTeams(ctx context.Context, id int32) ([]TournamentTeam, error) {
	var out []TournamentTeam
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentTeams, id), nil, &out)
}

func (c *Client) GetTournamentWinner(ctx context.Context, id int32) (*TournamentWinnerResponse, error) {
	var out TournamentWinnerResponse
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentWinner, id), nil, &out)
}

func (c *Client) GetTournamentStatus(ctx context.Context, id int32) (*FinishedResponse, error) {
	var out FinishedResponse
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1TournamentStatus, id), nil, &out)
}

// --- Matches ---

// SubmitScore records one hole score. A 204 (no body) is success.
func (c *Client) SubmitScore(ctx context.Context, matchID int32, req ScoreSubmission) error {
	return c.do(ctx, http.MethodPost, pathID(RouteV1MatchScores, matchID), req, nil)
}

func (c *Client) GetMatchScores(ctx context.Context, matchID int32) ([]HoleStatus, error) {
	var out []HoleStatus
	return out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchScores, matchID), nil, &out)
}

func (c *Client) GetMatchWinner(ctx context.Context, matchID int32) (*MatchWinnerResponse, error) {
	var out MatchWinnerResponse
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchWinner, matchID), nil, &out)
}

func (c *Client) GetMatchStatus(ctx context.Context, matchID int32) (*FinishedResponse, error) {
	var out FinishedResponse
	return &out, c.do(ctx, http.MethodGet, pathID(RouteV1MatchStatus, matchID), nil, &out)
}

// pathID substitutes the {id} path parameter in a route template.
func pathID(route string, id int32) string {
	return strings.Replace(route, "{id}", strconv.FormatInt(int64(id), 10), 1)
}

// do performs a request, attaching the bearer token, encoding req (if any) as JSON,
// and decoding a success body into result (if non-nil). Non-2xx responses become
// *APIError carrying the server's error message.
func (c *Client) do(ctx context.Context, method, route string, req, result any) error {
	var body io.Reader
	if req != nil {
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
