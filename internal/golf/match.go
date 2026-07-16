package golf

import "context"

// MatchService computes match play scoring. The real algorithm — format-aware
// (singles / fourball best-ball / one-ball team scores), team-attributable, with
// the handicap allocation convention — is implemented in Track A1 step 4 with TDD.
// These methods are placeholders so the package builds against the new schema.
type MatchService struct {
	MatchDB       matchDB
	ParticipantDB participantDB
	ScoreDB       scoreDB
	PlayerDB      playerDB
	Logger        logger
}

// CalculateMatchScores returns the hole-by-hole match progression.
func (s *MatchService) CalculateMatchScores(ctx context.Context, matchID int32) ([]HoleResult, error) {
	return []HoleResult{}, nil // TODO(step 4): compute from materialized results
}

// IsFinished reports whether the match is complete.
func (s *MatchService) IsFinished(ctx context.Context, matchID int32) (bool, error) {
	return false, nil // TODO(step 4)
}

// GetWinner returns the winning team's ID, or nil if the match is undecided.
func (s *MatchService) GetWinner(ctx context.Context, matchID int32) (*int32, error) {
	return nil, nil // TODO(step 4)
}
