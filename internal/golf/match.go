package golf

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// MatchService handles match-related business logic
type MatchService struct {
	MatchDB       matchDB
	ParticipantDB participantDB
	ScoreDB       scoreDB
	PlayerDB      playerDB
	Logger        logger
}

// HasStarted checks if a match has started by verifying all participants have scores.
//
// Algorithm:
//   - Return false if no participants
//   - Return false if any participant has no scores
//   - Return true if all participants have at least one score
func (s *MatchService) HasStarted(ctx context.Context, matchID int32) (bool, error) {
	participants, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("failed to list participants: %w", err)
	}

	if len(participants) == 0 {
		return false, nil
	}

	// Check if all participants have scores
	for _, participant := range participants {
		scores, err := s.ScoreDB.ListScoresByMatchAndPlayer(ctx, matchID, participant.PlayerID)
		if err != nil {
			return false, fmt.Errorf("failed to get scores for player %d: %w", participant.PlayerID, err)
		}
		if len(scores) == 0 {
			return false, nil
		}
	}

	return true, nil
}

// IsFinished checks if a match is complete either by all holes being played
// or by early termination (dormie).
//
// Algorithm:
//   - Return false if no participants
//   - Return true if last score has "&" in status text (dormie - match decided early)
//   - Return true if all participants have 18 scores
//   - Return false otherwise
func (s *MatchService) IsFinished(ctx context.Context, matchID int32) (bool, error) {
	participants, err := s.ParticipantDB.ListMatchParticipants(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("failed to list participants: %w", err)
	}

	if len(participants) == 0 {
		return false, nil
	}

	// Check if match decided early (dormie)
	matchScores, err := s.CalculateMatchScores(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("failed to calculate match scores: %w", err)
	}
	if len(matchScores) > 0 && strings.Contains(matchScores[len(matchScores)-1].StatusText, "&") {
		return true, nil
	}

	// Check if all participants have 18 scores
	for _, participant := range participants {
		scores, err := s.ScoreDB.ListScoresByMatchAndPlayer(ctx, matchID, participant.PlayerID)
		if err != nil {
			return false, fmt.Errorf("failed to get scores for player %d: %w", participant.PlayerID, err)
		}
		if len(scores) != 18 {
			return false, nil
		}
	}

	return true, nil
}

// GetWinner determines the winning team for a finished match.
//
// Algorithm:
//   - Return empty string if match not finished
//   - Check final match status (cumulative lead)
//   - Positive = Red wins, Negative = Blue wins, Zero = Tied
func (s *MatchService) GetWinner(ctx context.Context, matchID int32) (string, error) {
	finished, err := s.IsFinished(ctx, matchID)
	if err != nil {
		return "", fmt.Errorf("failed to check if match finished: %w", err)
	}
	if !finished {
		return "", ErrMatchNotFinished
	}

	matchScores, err := s.CalculateMatchScores(ctx, matchID)
	if err != nil {
		return "", fmt.Errorf("failed to calculate match scores: %w", err)
	}

	if len(matchScores) == 0 {
		return "", ErrMatchNotStarted
	}

	finalStatus := matchScores[len(matchScores)-1].MatchStatus
	if finalStatus > 0 {
		return TeamRed, nil
	} else if finalStatus < 0 {
		return TeamBlue, nil
	}
	return TeamTied, nil
}

// CalculateMatchScores computes hole-by-hole match progression using best ball scoring.
//
// This is the core match play algorithm.
//
// Steps:
//  1. Check if match started (early exit if not)
//  2. Get player-to-team mapping
//  3. Collect stroke scores per hole for each team member
//     - Use gross strokes if match.handicapped == false
//     - Use net strokes if match.handicapped == true
//  4. Calculate team score per hole = MINIMUM of team members' scores (best ball)
//  5. Loop through all holes:
//     - Calculate hole winner (lower score wins in golf)
//     - Update cumulative match status
//     - Generate status text based on lead and holes remaining
//  6. Return array of hole-by-hole match progression
func (s *MatchService) CalculateMatchScores(ctx context.Context, matchID int32) ([]MatchStatus, error) {
	// Check if match started
	started, err := s.HasStarted(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if match started: %w", err)
	}
	if !started {
		return []MatchStatus{}, nil
	}

	// Get match details
	match, err := s.MatchDB.GetMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}

	// Get player-to-team mapping
	playerTeams, err := s.ParticipantDB.ListMatchParticipantsWithTeam(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant teams: %w", err)
	}

	if len(playerTeams) == 0 {
		return nil, ErrNoParticipants
	}

	// Build player handicap map
	playerHandicaps := make(map[int32]float32)
	for playerID := range playerTeams {
		player, err := s.PlayerDB.GetPlayer(ctx, playerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get player %d: %w", playerID, err)
		}
		playerHandicaps[playerID] = player.Hdcp
	}

	// Get all scores
	scores, err := s.ScoreDB.ListScoresByMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scores: %w", err)
	}

	// Group scores by team and hole
	redScores := make(map[int32][]int32) // hole_number -> []strokes
	blueScores := make(map[int32][]int32)

	for _, score := range scores {
		var strokes int32
		if match.Handicapped {
			// Get hole handicap
			holeHdcp, err := s.ScoreDB.GetHoleHandicap(ctx, score.CourseID, score.TeeColorID, score.HoleNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to get hole handicap: %w", err)
			}
			strokes = CalculateNetStrokes(score.Strokes, playerHandicaps[score.PlayerID], holeHdcp)
		} else {
			strokes = score.Strokes
		}

		teamName := playerTeams[score.PlayerID]
		if teamName == TeamRed {
			redScores[score.HoleNumber] = append(redScores[score.HoleNumber], strokes)
		} else if teamName == TeamBlue {
			blueScores[score.HoleNumber] = append(blueScores[score.HoleNumber], strokes)
		}
	}

	// Calculate team scores per hole (minimum - best ball)
	var result []MatchStatus
	var cumulativeStatus int32 = 0
	var lastStatusText string = ""

	// Determine maximum hole number from scores
	maxHole := int32(0)
	for hole := range redScores {
		if hole > maxHole {
			maxHole = hole
		}
	}
	for hole := range blueScores {
		if hole > maxHole {
			maxHole = hole
		}
	}

	for hole := int32(1); hole <= maxHole; hole++ {
		redTeamScore := minSlice(redScores[hole])   // Best ball for Red
		blueTeamScore := minSlice(blueScores[hole]) // Best ball for Blue

		// Skip holes where either team hasn't posted a score
		if redTeamScore == 0 || blueTeamScore == 0 {
			continue
		}

		// Determine hole winner (lower score wins in golf)
		var statusChange int32
		if redTeamScore < blueTeamScore {
			statusChange = 1 // Red wins hole
		} else if blueTeamScore < redTeamScore {
			statusChange = -1 // Blue wins hole
		} else {
			statusChange = 0 // Halved
		}

		cumulativeStatus += statusChange
		holesRemaining := 18 - int(hole)

		// Generate status text
		statusText := generateStatusText(cumulativeStatus, holesRemaining, lastStatusText)
		lastStatusText = statusText

		result = append(result, MatchStatus{
			MatchStatus:   int(cumulativeStatus),
			StatusText:    statusText,
			RedTeamScore:  redTeamScore,
			BlueTeamScore: blueTeamScore,
		})

		// Early termination if match decided
		if strings.Contains(statusText, "&") {
			break
		}
	}

	return result, nil
}

// minSlice finds the minimum stroke count in a slice (best ball scoring)
func minSlice(scores []int32) int32 {
	if len(scores) == 0 {
		return 0
	}
	min := scores[0]
	for _, s := range scores[1:] {
		if s < min {
			min = s
		}
	}
	return min
}

// generateStatusText creates human-readable match status text
func generateStatusText(matchStatus int32, holesRemaining int, lastText string) string {
	// If already decided (dormie), keep status
	if strings.Contains(lastText, "&") {
		return lastText
	}

	// All square
	if matchStatus == 0 {
		return "AS"
	}

	lead := int(math.Abs(float64(matchStatus)))

	// Match decided (dormie) - lead is greater than holes remaining
	if lead > holesRemaining && holesRemaining > 0 {
		return fmt.Sprintf("%d & %d", lead, holesRemaining)
	}

	// Still playing
	return fmt.Sprintf("%d UP", lead)
}
