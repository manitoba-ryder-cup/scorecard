package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// Every sentinel, with the answer a caller gets. The specific sentinels wrap the generic
// ones, so a case placed below its generic would silently fall through to it — pinning the
// sentence, not just the status, is what makes that fail here.
var domainAnswers = map[string]struct {
	err     error
	status  int
	message string
}{
	"ErrInvalidInput":             {golf.ErrInvalidInput, http.StatusBadRequest, "That request wasn't valid."},
	"ErrConflict":                 {golf.ErrConflict, http.StatusConflict, "That conflicts with something that already exists."},
	"ErrNotFound":                 {golf.ErrNotFound, http.StatusNotFound, "Not found."},
	"ErrCourseNotFound":           {golf.ErrCourseNotFound, http.StatusNotFound, "Course not found."},
	"ErrMatchNotFound":            {golf.ErrMatchNotFound, http.StatusNotFound, "Match not found."},
	"ErrParticipantNotFound":      {golf.ErrParticipantNotFound, http.StatusNotFound, "That player isn't in this match."},
	"ErrPlayerNotFound":           {golf.ErrPlayerNotFound, http.StatusNotFound, "Player not found."},
	"ErrTeamMemberNotFound":       {golf.ErrTeamMemberNotFound, http.StatusNotFound, "That player isn't on this team."},
	"ErrTeamNotFound":             {golf.ErrTeamNotFound, http.StatusNotFound, "Team not found."},
	"ErrTournamentNotFound":       {golf.ErrTournamentNotFound, http.StatusNotFound, "Tournament not found."},
	"ErrTournamentPlayerNotFound": {golf.ErrTournamentPlayerNotFound, http.StatusNotFound, "That player isn't entered in this tournament."},
	"ErrScoredMatchDelete":        {golf.ErrScoredMatchDelete, http.StatusConflict, "That match has scores. Reset it before deleting it."},
	"ErrScoredMatchLineup":        {golf.ErrScoredMatchLineup, http.StatusConflict, "That match has scores. Reset it before changing its lineup."},
	"ErrScoredMatchTeeSet":        {golf.ErrScoredMatchTeeSet, http.StatusConflict, "That match has scores. Reset it before changing its tee set."},
	"ErrScoredPlayerUndraft":      {golf.ErrScoredPlayerUndraft, http.StatusConflict, "That player has been scored in a match. Reset it before undrafting them."},
}

func TestRespondDomainError_AnswersEverySentinel(t *testing.T) {
	for name, want := range domainAnswers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			// Wrapped, because a repository and a service each wrap before this is reached.
			respondDomainError(context.Background(), rec, fmt.Errorf("doing the thing: %w", want.err))
			if rec.Code != want.status {
				t.Errorf("want status %d, got %d", want.status, rec.Code)
			}
			if got := bodyOf(t, rec); got != want.message {
				t.Errorf("got %q, want %q", got, want.message)
			}
		})
	}
}

// A sentinel with no case of its own answers as its generic, which reads as a working 404
// while telling the caller nothing. The table above is only a guard if it stays complete.
func TestRespondDomainError_TableCoversEverySentinel(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "golf", "errors.go"))
	if err != nil {
		t.Fatalf("read domain errors: %v", err)
	}
	declared := regexp.MustCompile(`(?m)^\t(Err\w+)\s*=`).FindAllStringSubmatch(string(src), -1)
	if len(declared) == 0 {
		t.Fatal("no sentinels found; the declaration shape changed")
	}
	for _, m := range declared {
		if _, ok := domainAnswers[m[1]]; !ok {
			t.Errorf("golf.%s has no entry in domainAnswers, so nothing pins what a caller is told", m[1])
		}
	}
}

func TestRespondDomainError_UnknownErrorIsAFault(t *testing.T) {
	rec := httptest.NewRecorder()
	respondDomainError(context.Background(), rec, errors.New("connection reset"))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
	if got := bodyOf(t, rec); got != serverFault {
		t.Errorf("got %q, want the one fault message", got)
	}
}

func bodyOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body.Error
}

// A caller reads these. Reads are public, so the operation that failed would be told to
// any spectator, and it reads as a verdict on their request rather than a fault here.
func TestRespondError_CollapsesEveryServerFault(t *testing.T) {
	for _, message := range []string{"Failed to list players", "Failed to calculate match scores", "Failed to get teams data"} {
		rec := httptest.NewRecorder()
		respondError(context.Background(), rec, http.StatusInternalServerError, message, errors.New("connection reset"))
		if got := bodyOf(t, rec); got != serverFault {
			t.Errorf("%q reached the client as %q, want the one fault message", message, got)
		}
	}

	rec := httptest.NewRecorder()
	respondError(context.Background(), rec, http.StatusBadGateway, "Upstream is down", errors.New("boom"))
	if got := bodyOf(t, rec); got != serverFault {
		t.Errorf("502 reached the client as %q, want the one fault message", got)
	}
}

// A 4xx names something about the request, so it stays as the handler wrote it.
func TestRespondError_KeepsClientFacingMessages(t *testing.T) {
	for status, message := range map[int]string{
		http.StatusNotFound:   "Match not found",
		http.StatusBadRequest: "hole_number must be between 1 and 18",
		http.StatusConflict:   "Scoring is closed for this match",
	} {
		rec := httptest.NewRecorder()
		respondError(context.Background(), rec, status, message, errors.New("cause"))
		if got := bodyOf(t, rec); got != message {
			t.Errorf("status %d: got %q, want the message unchanged", status, got)
		}
	}
}

type unknownFieldReq struct {
	Name string `json:"name"`
}

func (unknownFieldReq) Validate(context.Context) error { return nil }

// A field the server does not know means the client believes it set something. Accepting
// the request and ignoring half of it is the worse answer.
func TestDecodeAndValidateRejectsUnknownFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/players", strings.NewReader(`{"name":"a","nmae":"typo"}`))

	if _, ok := decodeAndValidate[unknownFieldReq](rec, req); ok {
		t.Fatal("want a body with an unknown field refused")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDecodeAndValidateAcceptsAKnownBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/players", strings.NewReader(`{"name":"a"}`))

	got, ok := decodeAndValidate[unknownFieldReq](rec, req)
	if !ok {
		t.Fatalf("want an ordinary body accepted, got %d", rec.Code)
	}
	if got.Name != "a" {
		t.Errorf("decoded %+v", got)
	}
}
