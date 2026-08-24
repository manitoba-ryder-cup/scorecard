package golf

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// MatchService owns match setup (creating matches) and scoring (the live progression,
// and recomputing/persisting the materialized result on score writes).
type MatchService struct {
	MatchDB       matchDB
	ParticipantDB participantDB
	ScoreDB       scoreDB
	ResultDB      resultDB
	HoleDB        holeDB
	// Now is the clock the scoring window is read against; nil means time.Now.
	Now func() time.Time
}

func (s *MatchService) now() time.Time {
	if s.Now == nil {
		return time.Now()
	}
	return s.Now()
}

// CreateMatchInput is the intent to create a match within a tournament. The FK to
// tee_sets(course_id, tee_color_id) means the tee must be configured for the course.
// TeeTime is required: it is what the scoring window is measured from. Request-shape
// validation is at the API boundary; unknown references surface as ErrInvalidInput
// from the repository.
type CreateMatchInput struct {
	TournamentID  uuid.UUID
	CourseID      uuid.UUID
	TeeColorID    uuid.UUID
	MatchFormatID uuid.UUID
	TeeTime       time.Time
	Handicapped   bool
}

// CreateMatch persists a new match.
func (s *MatchService) CreateMatch(ctx context.Context, in CreateMatchInput) (*Match, error) {
	match, err := s.MatchDB.CreateMatch(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to create match: %w", err)
	}
	return match, nil
}

// UpdateMatchInput changes an existing match. The tournament is deliberately not among
// the fields: a match's scores and participants reference it, so moving one between cups
// is not an edit.
type UpdateMatchInput struct {
	ID            uuid.UUID
	CourseID      *uuid.UUID
	TeeColorID    *uuid.UUID
	MatchFormatID *uuid.UUID
	TeeTime       *time.Time
	Handicapped   *bool
}

// movesTeeSet reports whether in changes the course or tee colour. Re-sending the values a
// match already has is not a move.
func (in UpdateMatchInput) movesTeeSet(current Match) bool {
	return (in.CourseID != nil && *in.CourseID != current.CourseID) ||
		(in.TeeColorID != nil && *in.TeeColorID != current.TeeColorID)
}

// UpdateMatch is deliberately allowed on a match that already has scores: the case that
// needs it most is a group that went out late with a hole already entered, and the scoring
// window is measured from the tee time on every submission. The tee set is the exception —
// scores read their par and stroke index from it, so moving a scored match would leave them
// describing a round nobody played.
func (s *MatchService) UpdateMatch(ctx context.Context, in UpdateMatchInput) (*Match, error) {
	match, err := s.MatchDB.UpdateMatch(ctx, in, func(current Match, scored bool) error {
		if scored && in.movesTeeSet(current) {
			return fmt.Errorf("%w: match %s", ErrScoredMatchTeeSet, in.ID)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update match: %w", err)
	}
	return match, nil
}

// ListMatches returns a tournament's matches.
func (s *MatchService) ListMatches(ctx context.Context, tournamentID uuid.UUID) ([]Match, error) {
	matches, err := s.MatchDB.ListMatchesByTournament(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list matches: %w", err)
	}
	return matches, nil
}

// AddParticipant adds a player (on a team) to a match. The match is loaded first (so a
// bad match is a clean 404) to derive the tournament. The composite FKs enforce that
// the player is drafted onto that team and the team is in the match's tournament — an
// undrafted or wrong-team player surfaces as ErrInvalidInput.
func (s *MatchService) AddParticipant(ctx context.Context, matchID, playerID, teamID uuid.UUID) (*MatchParticipant, error) {
	match, err := s.MatchDB.GetMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to load match: %w", err)
	}
	participant, err := s.ParticipantDB.CreateMatchParticipant(ctx, match.TournamentID, matchID, playerID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}
	return participant, nil
}

// RemoveParticipant removes a player from a match. ErrParticipantNotFound if they weren't in
// it, and a scored match is refused — its scores are recorded against the lineup that played.
func (s *MatchService) RemoveParticipant(ctx context.Context, matchID, playerID uuid.UUID) error {
	err := s.ParticipantDB.DeleteMatchParticipant(ctx, matchID, playerID, func(scored bool) error {
		if scored {
			return fmt.Errorf("%w: match %s", ErrScoredMatchLineup, matchID)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	return nil
}

// ListParticipants returns a match's participants.
func (s *MatchService) ListParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error) {
	participants, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}
	return participants, nil
}

// ScoreEntry is one competitor's client-supplied score on a hole. CourseID/TeeColorID
// are intentionally absent — they're derived from the match, not trusted from the caller;
// the hole is a parameter, because scores are recorded a hole at a time. PlayerID is nil
// for one-ball team formats (alt shot, scramble). Strokes range is validated at the API
// boundary; the team-in-match invariant below needs match state.
type ScoreEntry struct {
	TeamID   uuid.UUID
	PlayerID *uuid.UUID
	Strokes  int32
}

// A match's scores can be recorded from shortly before it tees off until well after it
// could still be under way. Both bounds are relative to the tee time, which is an
// instant, so the window needs no timezone.
const (
	// Enough to cover a group moved up, without opening the day before. A delay needs no
	// allowance: the window is already open by then.
	scoringOpensBefore = 2 * time.Hour
	// A round is 3.5–5 hours, so this leaves hours to spare for corrections after the
	// last putt drops, and still shuts the same day.
	scoringClosesAfter = 12 * time.Hour
)

// ScoringWindow returns the interval a match's scores can be recorded in.
//
// Published on every match because a client holding its own copy of the rule put it in two
// repos with nothing keeping them equal. Bounds rather than a yes/no: a boolean is true only
// at the instant it was computed, and this page is left open on a fairway for hours.
func ScoringWindow(teeTime time.Time) (opens, closes time.Time) {
	return teeTime.Add(-scoringOpensBefore), teeTime.Add(scoringClosesAfter)
}

// scoringOpen reports whether a match's scores can be recorded at the given moment. This
// is the copy that decides; the client's gate is only there to avoid offering a control
// that cannot work.
//
// Deliberately not tied to the tournament's dates: those are a calendar day wide, which
// let Sunday's matches be scored on Saturday and only mean anything in some timezone.
func scoringOpen(now, teeTime time.Time) bool {
	opens, closes := ScoringWindow(teeTime)
	return !now.Before(opens) && !now.After(closes)
}

// SubmitHoleScores persists every score for one hole and recomputes the match's
// materialized result — the single write path that keeps match_results in sync. The hole
// is written as a unit, so a caller cannot leave one side scored and the other not. It
// returns the recomputed result, so a caller learns the hole closed the match out without
// having to re-derive the close-out rule.
func (s *MatchService) SubmitHoleScores(ctx context.Context, matchID uuid.UUID, hole int32, entries []ScoreEntry) (StoredResult, error) {
	var zero StoredResult
	match, err := s.MatchDB.GetMatch(ctx, matchID)
	if err != nil {
		return zero, fmt.Errorf("failed to get match: %w", err)
	}
	// A match months out is being poked at, and one from a past cup is history.
	if !scoringOpen(s.now(), match.TeeTime) {
		opens, closes := ScoringWindow(match.TeeTime)
		return zero, fmt.Errorf("%w: match %s tees off at %s; scores can only be recorded from %s to %s",
			ErrConflict, matchID, match.TeeTime.Format(time.RFC3339),
			opens.Format(time.RFC3339), closes.Format(time.RFC3339))
	}

	// Needs the match's participants, so it is a domain invariant rather than shape.
	teamA, teamB, ok, err := s.matchTeams(ctx, matchID)
	if err != nil {
		return zero, err
	}
	scores := make([]Score, len(entries))
	for i, entry := range entries {
		if !ok || (entry.TeamID != teamA && entry.TeamID != teamB) {
			return zero, fmt.Errorf("%w: team %s is not in match %s", ErrInvalidInput, entry.TeamID, matchID)
		}
		// Course/tee come from the match — the score's holes constraint keys off them.
		scores[i] = Score{
			MatchID:    matchID,
			TeamID:     entry.TeamID,
			PlayerID:   entry.PlayerID,
			CourseID:   match.CourseID,
			TeeColorID: match.TeeColorID,
			HoleNumber: hole,
			Strokes:    entry.Strokes,
		}
	}

	// Handed to the repo so the check and the writes share a transaction; here they race.
	guard := func(before []Score) error {
		if ComputeStoredResult(before, teamA, teamB).Finished && !holeIsScored(before, hole) {
			return fmt.Errorf("%w: match %s is complete; hole %d cannot be scored", ErrConflict, matchID, hole)
		}
		return nil
	}
	recompute := func(after []Score) StoredResult {
		return ComputeStoredResult(after, teamA, teamB)
	}
	result, err := s.ScoreDB.SaveScoresAndRecompute(ctx, matchID, scores, match.TournamentID, guard, recompute)
	if err != nil {
		return zero, fmt.Errorf("failed to save scores: %w", err)
	}
	return result, nil
}

// holeIsScored reports whether a hole already carries any score, which is what separates
// correcting a played hole from adding one.
func holeIsScored(scores []Score, hole int32) bool {
	return slices.ContainsFunc(scores, func(s Score) bool { return s.HoleNumber == hole })
}

// CalculateMatchScores computes the live hole-by-hole match-play progression.
func (s *MatchService) CalculateMatchScores(ctx context.Context, matchID uuid.UUID) ([]HoleResult, error) {
	teamA, teamB, ok, err := s.matchTeams(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []HoleResult{}, nil
	}
	scores, err := s.ScoreDB.ListScoresByMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scores: %w", err)
	}
	return ComputeMatchProgress(scores, teamA, teamB), nil
}

// ListMatchHoles returns the holes of a match's tee set. The match is loaded first so
// a bad match id is a clean 404 rather than an empty list.
func (s *MatchService) ListMatchHoles(ctx context.Context, matchID uuid.UUID) ([]Hole, error) {
	match, err := s.MatchDB.GetMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}
	holes, err := s.HoleDB.ListHolesByTeeSet(ctx, match.CourseID, match.TeeColorID)
	if err != nil {
		return nil, fmt.Errorf("failed to list holes: %w", err)
	}
	return holes, nil
}

// DeleteMatch removes a match and its lineup. A scored match is refused — losing results is a
// decision, not a side effect of tidying up.
func (s *MatchService) DeleteMatch(ctx context.Context, matchID uuid.UUID) error {
	err := s.MatchDB.DeleteMatch(ctx, matchID, func(scored bool) error {
		if scored {
			return fmt.Errorf("%w: match %s", ErrScoredMatchDelete, matchID)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete match: %w", err)
	}
	return nil
}

// ResetMatch clears a match's scores and stored result, leaving its lineup.
func (s *MatchService) ResetMatch(ctx context.Context, matchID uuid.UUID) error {
	if err := s.ScoreDB.ResetMatch(ctx, matchID); err != nil {
		return fmt.Errorf("failed to reset match: %w", err)
	}
	return nil
}

// ListResults builds every match's outcome for a tournament, plus where the cup stands.
// Participants and scores are fetched tournament-wide and grouped by match, so the view
// costs a fixed number of queries however many matches there are.
//
// The phase comes from the stored results rather than the scores in hand, because the other
// endpoints publishing it read those and the two can disagree.
func (s *MatchService) ListResults(ctx context.Context, tournamentID uuid.UUID) ([]MatchResult, sdk.TournamentPhase, error) {
	matches, err := s.MatchDB.ListMatchDetailsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, sdk.PhaseUpcoming, fmt.Errorf("failed to list matches: %w", err)
	}
	participants, err := s.ParticipantDB.ListParticipantsWithPlayersByTournament(ctx, tournamentID)
	if err != nil {
		return nil, sdk.PhaseUpcoming, fmt.Errorf("failed to list participants: %w", err)
	}
	scores, err := s.ScoreDB.ListScoresByTournament(ctx, tournamentID)
	if err != nil {
		return nil, sdk.PhaseUpcoming, fmt.Errorf("failed to list scores: %w", err)
	}
	outcomes, err := s.ResultDB.ListMatchOutcomes(ctx, tournamentID)
	if err != nil {
		return nil, sdk.PhaseUpcoming, fmt.Errorf("failed to list match outcomes: %w", err)
	}

	sidesByMatch := groupSides(participants)
	scoresByMatch := map[uuid.UUID][]Score{}
	for _, sc := range scores {
		scoresByMatch[sc.MatchID] = append(scoresByMatch[sc.MatchID], sc)
	}

	results := make([]MatchResult, len(matches))
	for i, m := range matches {
		results[i] = buildMatchResult(m, sidesByMatch[m.ID], scoresByMatch[m.ID])
	}
	return results, ComputePhase(outcomes), nil
}

// groupSides collapses a tournament's participants into each match's sides, preserving
// the query's ordering (by team, then player name) so a match's two sides and their
// pairings are stable.
func groupSides(participants []MatchParticipantPlayer) map[uuid.UUID][]MatchSide {
	sides := map[uuid.UUID][]MatchSide{}
	for _, p := range participants {
		matchSides := sides[p.MatchID]
		idx := -1
		for j := range matchSides {
			if matchSides[j].TeamID == p.TeamID {
				idx = j
				break
			}
		}
		if idx == -1 {
			matchSides = append(matchSides, MatchSide{TeamID: p.TeamID})
			idx = len(matchSides) - 1
		}
		matchSides[idx].Players = append(matchSides[idx].Players, MatchSidePlayer{
			PlayerID:  p.PlayerID,
			FirstName: p.FirstName,
			LastName:  p.LastName,
		})
		sides[p.MatchID] = matchSides
	}
	return sides
}

// buildMatchResult assembles one match's result. The two teams for scoring come from
// its sides; with fewer than two sides the match hasn't a scored progression yet, so it
// reads as unplayed.
func buildMatchResult(m MatchDetail, sides []MatchSide, scores []Score) MatchResult {
	result := MatchResult{
		MatchID:      m.ID,
		FormatName:   m.FormatName,
		CourseName:   m.CourseName,
		TeeTime:      m.TeeTime,
		Sides:        sides,
		StoredResult: StoredResult{HolesRemaining: 18},
		HoleResults:  []*uuid.UUID{},
	}
	if len(sides) != 2 {
		return result
	}

	teamA, teamB := sides[0].TeamID, sides[1].TeamID
	progress := ComputeMatchProgress(scores, teamA, teamB)
	holeResults := make([]*uuid.UUID, len(progress))
	for i, h := range progress {
		holeResults[i] = HoleWinner(h)
	}
	result.HoleResults = holeResults

	result.StoredResult = ComputeStoredResult(scores, teamA, teamB)
	return result
}

// MatchStatus reads a match's stored outcome — the same state a score write returns, so
// polling it and recording a score describe the match identically. An unplayed match has
// no stored row and reads as the zero result.
func (s *MatchService) MatchStatus(ctx context.Context, matchID uuid.UUID) (StoredResult, error) {
	r, err := s.ResultDB.GetMatchResult(ctx, matchID)
	if err != nil {
		return StoredResult{}, fmt.Errorf("failed to get match result: %w", err)
	}
	if r == nil {
		return StoredResult{HolesRemaining: 18}, nil
	}
	return *r, nil
}

// matchTeams returns the two distinct team IDs among a match's participants.
func (s *MatchService) matchTeams(ctx context.Context, matchID uuid.UUID) (uuid.UUID, uuid.UUID, bool, error) {
	ps, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("failed to list participants: %w", err)
	}
	var a, b uuid.UUID
	for _, p := range ps {
		switch {
		case a == uuid.Nil || p.TeamID == a:
			a = p.TeamID
		case b == uuid.Nil || p.TeamID == b:
			b = p.TeamID
		default:
			return uuid.Nil, uuid.Nil, false, nil // a third side is not a match
		}
	}
	if a == uuid.Nil || b == uuid.Nil {
		return uuid.Nil, uuid.Nil, false, nil
	}
	return a, b, true, nil
}
