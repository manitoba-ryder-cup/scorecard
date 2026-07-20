package golf

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MatchService owns match setup (creating matches) and scoring (the live progression,
// and recomputing/persisting the materialized result on score writes).
type MatchService struct {
	MatchDB       matchDB
	ParticipantDB participantDB
	ScoreDB       scoreDB
	ResultDB      resultDB
}

// CreateMatchInput is the intent to create a match within a tournament. The FK to
// tee_sets(course_id, tee_color_id) means the tee must be configured for the course.
// TeeTime is optional (an unscheduled match). Request-shape validation is at the API
// boundary; unknown references surface as ErrInvalidInput from the repository.
type CreateMatchInput struct {
	TournamentID  uuid.UUID
	CourseID      uuid.UUID
	TeeColorID    uuid.UUID
	MatchFormatID uuid.UUID
	TeeTime       *time.Time
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

// ListParticipants returns a match's participants.
func (s *MatchService) ListParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error) {
	participants, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}
	return participants, nil
}

// ScoreEntry is a client-supplied hole score. CourseID/TeeColorID are intentionally
// absent — they're derived from the match, not trusted from the caller. PlayerID is
// nil for one-ball team formats (alt shot, scramble). Hole/strokes range is validated
// at the API boundary; the team-in-match invariant below needs match state.
type ScoreEntry struct {
	HoleNumber int32
	Strokes    int32
	TeamID     uuid.UUID
	PlayerID   *uuid.UUID
}

// SubmitScore persists one hole score, then recomputes the match's materialized
// result — the single write path that keeps match_results in sync.
func (s *MatchService) SubmitScore(ctx context.Context, matchID uuid.UUID, entry ScoreEntry) error {
	match, err := s.MatchDB.GetMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}
	// Reject scores for a team that isn't actually playing this match. This needs the
	// match's participants, so it's a domain invariant, not boundary shape validation.
	teamA, teamB, ok, err := s.matchTeams(ctx, matchID)
	if err != nil {
		return err
	}
	if !ok || (entry.TeamID != teamA && entry.TeamID != teamB) {
		return fmt.Errorf("%w: team %s is not in match %s", ErrInvalidInput, entry.TeamID, matchID)
	}

	// Course/tee come from the match — the score's holes constraint keys off them.
	score := Score{
		MatchID:    matchID,
		TeamID:     entry.TeamID,
		PlayerID:   entry.PlayerID,
		CourseID:   match.CourseID,
		TeeColorID: match.TeeColorID,
		HoleNumber: entry.HoleNumber,
		Strokes:    entry.Strokes,
	}
	if err := s.ScoreDB.SaveScore(ctx, score); err != nil {
		return fmt.Errorf("failed to save score: %w", err)
	}
	// Reuse the match/teams already loaded above rather than re-fetching them.
	return s.recompute(ctx, matchID, match.TournamentID, teamA, teamB)
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

// recompute recomputes a match's materialized result from its scores and upserts it to
// match_results, given the already-resolved tournament and two teams. Kept separate so
// the hot SubmitScore path can pass values it already loaded instead of re-fetching the
// match and participants.
func (s *MatchService) recompute(ctx context.Context, matchID, tournamentID, teamA, teamB uuid.UUID) error {
	scores, err := s.ScoreDB.ListScoresByMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to list scores: %w", err)
	}
	result := ComputeStoredResult(scores, teamA, teamB)
	return s.ResultDB.UpsertMatchResult(ctx, matchID, tournamentID, result)
}

// IsFinished reports whether the match is complete, from the stored result.
func (s *MatchService) IsFinished(ctx context.Context, matchID uuid.UUID) (bool, error) {
	r, err := s.ResultDB.GetMatchResult(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("failed to get match result: %w", err)
	}
	return r != nil && r.Finished, nil
}

// GetWinner returns the winning team's ID, or nil if the match is undecided.
func (s *MatchService) GetWinner(ctx context.Context, matchID uuid.UUID) (*uuid.UUID, error) {
	r, err := s.ResultDB.GetMatchResult(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match result: %w", err)
	}
	if r == nil || !r.Finished {
		return nil, nil
	}
	return r.LeaderTeamID, nil
}

// matchTeams returns the two distinct team IDs among a match's participants.
func (s *MatchService) matchTeams(ctx context.Context, matchID uuid.UUID) (uuid.UUID, uuid.UUID, bool, error) {
	ps, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("failed to list participants: %w", err)
	}
	var ids []uuid.UUID
	for _, p := range ps {
		seen := false
		for _, id := range ids {
			if id == p.TeamID {
				seen = true
			}
		}
		if !seen {
			ids = append(ids, p.TeamID)
		}
	}
	if len(ids) != 2 {
		return uuid.Nil, uuid.Nil, false, nil
	}
	return ids[0], ids[1], true, nil
}
