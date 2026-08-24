package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// maxRequestBody caps decoded request bodies to guard against a client streaming an
// unbounded payload into memory.
const maxRequestBody = 1 << 20 // 1 MiB

// validatable is implemented by the SDK request types (Validate runs client-side too).
type validatable interface {
	Validate(ctx context.Context) error
}

// decodeAndValidate reads a size-limited JSON body into a T and validates its shape,
// writing a 400 and returning ok=false on any failure. It collapses the identical
// decode -> validate -> respond preamble every write handler shared.
func decodeAndValidate[T validatable](w http.ResponseWriter, req *http.Request) (T, bool) {
	var body T
	req.Body = http.MaxBytesReader(w, req.Body, maxRequestBody)
	decoder := json.NewDecoder(req.Body)
	// A field the server does not know is a client that thinks it set something. Saying so
	// beats accepting the request and quietly ignoring half of it.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		respondError(req.Context(), w, http.StatusBadRequest, "Invalid request body", err)
		return body, false
	}
	if err := body.Validate(req.Context()); err != nil {
		respondError(req.Context(), w, http.StatusBadRequest, err.Error(), nil)
		return body, false
	}
	return body, true
}

// respondJSON sends a JSON response
func respondJSON(writer http.ResponseWriter, status int, data any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err, "status", status)
	}
}

// serverFault is all a caller is told when the fault is ours. The operation that failed is
// worth having in the log and misleading on the wire, where it reads as a verdict on the
// request rather than a fault here.
const serverFault = "Sorry, something went wrong. Please try again later."

// respondError sends the SDK's error envelope. A 4xx logs at Warn so a bad request does not
// pollute the error stream, and the *Context variants carry the request's tenant, actor and
// request id along with it.
func respondError(ctx context.Context, writer http.ResponseWriter, status int, message string, err error) {
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(ctx, "API error", "message", message, "error", err, "status", status)
		message = serverFault
	} else if err != nil {
		slog.WarnContext(ctx, "API client error", "message", message, "error", err, "status", status)
	}
	respondJSON(writer, status, sdk.ErrorResponse{Error: message})
}

// domainAnswers is ordered, and the first sentinel that matches wins. A specific sentinel
// wraps its generic, so each must sit above the generic it wraps or it never answers for
// itself. Order is the load-bearing part of this list, not an accident of how it was typed.
var domainAnswers = []struct {
	err     error
	status  int
	message string
}{
	{golf.ErrCourseNotFound, http.StatusNotFound, "Course not found."},
	{golf.ErrMatchNotFound, http.StatusNotFound, "Match not found."},
	{golf.ErrParticipantNotFound, http.StatusNotFound, "That player isn't in this match."},
	{golf.ErrPlayerNotFound, http.StatusNotFound, "Player not found."},
	{golf.ErrTeamMemberNotFound, http.StatusNotFound, "That player isn't on this team."},
	{golf.ErrTeamNotFound, http.StatusNotFound, "Team not found."},
	{golf.ErrTournamentNotFound, http.StatusNotFound, "Tournament not found."},
	{golf.ErrTournamentPlayerNotFound, http.StatusNotFound, "That player isn't entered in this tournament."},

	{golf.ErrScoredMatchDelete, http.StatusConflict, "That match has scores. Reset it before deleting it."},
	{golf.ErrScoredMatchLineup, http.StatusConflict, "That match has scores. Reset it before changing its lineup."},
	{golf.ErrScoredMatchTeeSet, http.StatusConflict, "That match has scores. Reset it before changing its tee set."},
	{golf.ErrScoredPlayerUndraft, http.StatusConflict, "That player has been scored in a match. Reset it before undrafting them."},

	{golf.ErrNotFound, http.StatusNotFound, "Not found."},
	{golf.ErrInvalidInput, http.StatusBadRequest, "That request wasn't valid."},
	{golf.ErrConflict, http.StatusConflict, "That conflicts with something that already exists."},
}

// respondDomainError answers a domain failure. The sentinel decides both the status and the
// sentence, so the wording lives here rather than at each call site, and the operation that
// failed is already in the error's wrapped chain, which is what reaches the log.
func respondDomainError(ctx context.Context, writer http.ResponseWriter, err error) {
	for _, answer := range domainAnswers {
		if errors.Is(err, answer.err) {
			respondError(ctx, writer, answer.status, answer.message, err)
			return
		}
	}
	respondError(ctx, writer, http.StatusInternalServerError, "Request failed", err)
}
