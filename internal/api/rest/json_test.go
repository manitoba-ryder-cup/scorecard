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
)

// Order is what the lookup runs on, so this walks the real table and asks each entry to
// answer for itself. A specific sentinel sitting below the generic it wraps answers as the
// generic, and the entry's own message is what no longer comes back.
func TestRespondDomainError_AnswersEverySentinel(t *testing.T) {
	for _, answer := range domainAnswers {
		t.Run(answer.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			// Wrapped, because a repository and a service each wrap before this is reached.
			respondDomainError(context.Background(), rec, fmt.Errorf("doing the thing: %w", answer.err))
			if rec.Code != answer.status {
				t.Errorf("want status %d, got %d", answer.status, rec.Code)
			}
			if got := bodyOf(t, rec); got != answer.message {
				t.Errorf("got %q, want %q", got, answer.message)
			}
		})
	}
}

// A sentinel with no row of its own answers as its generic, which reads as a working 404
// while telling the caller nothing. Go cannot enumerate a package's vars, so the declarations
// are read from source and matched by name against the rows meant to cover them.
func TestRespondDomainError_TableCoversEverySentinel(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "golf", "errors.go"))
	if err != nil {
		t.Fatalf("read domain errors: %v", err)
	}
	// Both shapes a sentinel is declared in: grouped in a var block, or on its own.
	declared := regexp.MustCompile(`(?m)^(?:\t|var\s+)(Err\w+)\s*=`).FindAllStringSubmatch(string(src), -1)
	if len(declared) == 0 {
		t.Fatal("no sentinels found; the declaration shape changed")
	}
	covered := map[string]bool{}
	for _, answer := range domainAnswers {
		covered[answer.name] = true
	}
	for _, m := range declared {
		if !covered[m[1]] {
			t.Errorf("golf.%s has no row in domainAnswers, so it answers as its generic and nothing pins it", m[1])
		}
		delete(covered, m[1])
	}
	for name := range covered {
		t.Errorf("domainAnswers has a row for %q, which the domain does not declare", name)
	}
}

// A row naming one sentinel while pointing at another passes the coverage check by name, and
// leaves the sentinel it claims to cover unreached. Two rows on one error is that mistake.
func TestRespondDomainError_NoSentinelIsAnsweredTwice(t *testing.T) {
	seen := map[error]string{}
	for _, answer := range domainAnswers {
		if first, dup := seen[answer.err]; dup {
			t.Errorf("%s and %s are rows on the same error; the second can never be reached", first, answer.name)
		}
		seen[answer.err] = answer.name
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
