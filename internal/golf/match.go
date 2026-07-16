package golf

import (
	"context"
	"fmt"
)

// MatchService owns match scoring: computing the live progression, and recomputing
// and persisting the materialized result on score writes.
type MatchService struct {
	MatchDB       matchDB
	ParticipantDB participantDB
	ScoreDB       scoreDB
	ResultDB      resultDB
	Logger        logger
}

// CalculateMatchScores computes the live hole-by-hole match-play progression.
func (s *MatchService) CalculateMatchScores(ctx context.Context, matchID int32) ([]HoleResult, error) {
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

// RecomputeResult recomputes a match's result from its scores and persists it to
// match_results. Called after any score write for the match.
func (s *MatchService) RecomputeResult(ctx context.Context, matchID int32) error {
	match, err := s.MatchDB.GetMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}
	teamA, teamB, ok, err := s.matchTeams(ctx, matchID)
	if err != nil {
		return err
	}
	if !ok {
		return nil // fewer than two teams present; nothing to materialize yet
	}
	scores, err := s.ScoreDB.ListScoresByMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to list scores: %w", err)
	}
	result := ComputeStoredResult(scores, teamA, teamB)
	return s.ResultDB.UpsertMatchResult(ctx, matchID, match.TournamentID, result)
}

// IsFinished reports whether the match is complete, from the stored result.
func (s *MatchService) IsFinished(ctx context.Context, matchID int32) (bool, error) {
	r, err := s.ResultDB.GetMatchResult(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("failed to get match result: %w", err)
	}
	return r != nil && r.Finished, nil
}

// GetWinner returns the winning team's ID, or nil if the match is undecided.
func (s *MatchService) GetWinner(ctx context.Context, matchID int32) (*int32, error) {
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
func (s *MatchService) matchTeams(ctx context.Context, matchID int32) (int32, int32, bool, error) {
	ps, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to list participants: %w", err)
	}
	var ids []int32
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
		return 0, 0, false, nil
	}
	return ids[0], ids[1], true, nil
}
