package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// Records what the handler asked for and answers with a match, so a test can assert on
// the input rather than on a round trip through a database.
type recordingMatchService struct {
	MatchService
	got golf.UpdateMatchInput
}

func (s *recordingMatchService) UpdateMatch(ctx context.Context, in golf.UpdateMatchInput) (*golf.Match, error) {
	s.got = in
	return &golf.Match{ID: in.ID}, nil
}

func putMatch(t *testing.T, body string) (*recordingMatchService, *httptest.ResponseRecorder) {
	t.Helper()
	svc := &recordingMatchService{}
	req := httptest.NewRequest("PUT", "/v1/matches/"+matchID.String(), strings.NewReader(body))
	req.SetPathValue("id", matchID.String())
	rec := httptest.NewRecorder()
	NewMatchesHandler(svc).UpdateMatch(rec, req)
	return svc, rec
}

var matchID = uuid.New()

// The property the whole design rests on: a field the caller left out arrives nil, so the
// repository's COALESCE leaves the stored column alone. If these ever came through as
// zero values, a tee-time edit would silently blank the course and format.
func TestUpdateMatch_OmittedFieldsStayUnset(t *testing.T) {
	svc, rec := putMatch(t, `{"tee_time":"2026-09-18T13:00:00Z"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if svc.got.ID != matchID {
		t.Errorf("id = %s, want %s", svc.got.ID, matchID)
	}
	if svc.got.TeeTime == nil || !svc.got.TeeTime.Equal(time.Date(2026, 9, 18, 13, 0, 0, 0, time.UTC)) {
		t.Errorf("tee_time = %v, want 2026-09-18T13:00:00Z", svc.got.TeeTime)
	}
	if svc.got.CourseID != nil || svc.got.TeeColorID != nil || svc.got.MatchFormatID != nil || svc.got.Handicapped != nil {
		t.Errorf("unmentioned fields should be nil, got %+v", svc.got)
	}
}

// false is a real value. Read as "absent" it would be impossible to turn handicapping off.
func TestUpdateMatch_HandicappedFalseIsSet(t *testing.T) {
	svc, rec := putMatch(t, `{"handicapped":false}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if svc.got.Handicapped == nil || *svc.got.Handicapped {
		t.Errorf("handicapped = %v, want a set false", svc.got.Handicapped)
	}
}

func TestUpdateMatch_RejectsAnEmptyBody(t *testing.T) {
	_, rec := putMatch(t, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateMatch_RejectsABadID(t *testing.T) {
	svc := &recordingMatchService{}
	req := httptest.NewRequest("PUT", "/v1/matches/nope", strings.NewReader(`{"handicapped":true}`))
	req.SetPathValue("id", "nope")
	rec := httptest.NewRecorder()
	NewMatchesHandler(svc).UpdateMatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateMatch_RespondsWithTheMatch(t *testing.T) {
	_, rec := putMatch(t, `{"tee_time":"2026-09-18T13:00:00Z"}`)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got["id"] != matchID.String() {
		t.Errorf("id = %v, want %s", got["id"], matchID)
	}
}
