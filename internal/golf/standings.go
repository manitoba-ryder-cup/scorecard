package golf

import (
	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// Ryder Cup scoring: a won match is worth a full point to the winning side, a halved
// match half a point to each.
const (
	pointsForWin   = 1.0
	pointsForHalve = 0.5
)

// ComputeTeamPoints tallies a tournament's points for each of its teams. Every team gets
// an entry, so a side that has won nothing reports zero rather than being absent.
func ComputeTeamPoints(outcomes []MatchOutcome, teamIDs []uuid.UUID) map[uuid.UUID]float64 {
	points := make(map[uuid.UUID]float64, len(teamIDs))
	for _, id := range teamIDs {
		points[id] = 0
	}
	for _, o := range outcomes {
		switch {
		case !o.Finished:
			continue
		case o.WinnerTeamID == nil:
			for _, id := range teamIDs {
				points[id] += pointsForHalve
			}
		default:
			points[*o.WinnerTeamID] += pointsForWin
		}
	}
	return points
}

// IsTournamentComplete reports whether every match has a final result. A tournament with
// no matches has not been played, so it is not complete — otherwise a freshly created Cup
// would report itself finished.
func IsTournamentComplete(outcomes []MatchOutcome) bool {
	if len(outcomes) == 0 {
		return false
	}
	for _, o := range outcomes {
		if !o.Finished {
			return false
		}
	}
	return true
}

// ComputePhase places a tournament by its matches: live from the first score recorded
// until the last match is final.
func ComputePhase(outcomes []MatchOutcome) sdk.TournamentPhase {
	switch {
	case IsTournamentComplete(outcomes):
		return sdk.PhaseFinished
	case anyStarted(outcomes):
		return sdk.PhaseLive
	default:
		return sdk.PhaseUpcoming
	}
}

func anyStarted(outcomes []MatchOutcome) bool {
	for _, o := range outcomes {
		if o.Started {
			return true
		}
	}
	return false
}

// ComputeTournamentOutcome decides the Cup: it is settled only once every match is final
// (no early clinch), and won by the sole points leader. A tie leaves no winner. There is
// no holder to retain it — the captains change every year — and the tie-break (the last,
// most tightly contested group goes back out) has never been needed, so it is not
// modelled.
func ComputeTournamentOutcome(outcomes []MatchOutcome, teamIDs []uuid.UUID) TournamentOutcome {
	if !IsTournamentComplete(outcomes) {
		return TournamentOutcome{}
	}

	points := ComputeTeamPoints(outcomes, teamIDs)
	var leader *uuid.UUID
	var best, runnerUp float64
	for _, id := range teamIDs {
		if p := points[id]; leader == nil || p > best {
			runnerUp = best
			best, leader = p, &id
		} else if p > runnerUp {
			runnerUp = p
		}
	}
	if leader == nil || best == runnerUp {
		return TournamentOutcome{Finished: true}
	}
	return TournamentOutcome{Finished: true, WinnerTeamID: leader}
}

// TournamentStandings is the raw material a tournament's verdict is computed from: the
// sides contesting it, and the outcome of each of its matches.
type TournamentStandings struct {
	TeamIDs  []uuid.UUID
	Outcomes []MatchOutcome
}

// TeamMembership is a player's side in one tournament — the draft record, which is what
// links a Cup win to the players who won it.
type TeamMembership struct {
	PlayerID     uuid.UUID
	TournamentID uuid.UUID
	TeamID       uuid.UUID
}

// CupData is the raw material for deciding Cup wins: every tournament's standings, and
// every player's side in each. Fetched together so counting Cups costs one round trip.
type CupData struct {
	Standings   map[uuid.UUID]TournamentStandings
	Memberships []TeamMembership
}

// ComputeCupsWon counts, per player, the finished tournaments their side won outright.
// A tied Cup counts for nobody.
func ComputeCupsWon(data CupData) map[uuid.UUID]int {
	winners := make(map[uuid.UUID]uuid.UUID, len(data.Standings))
	for tournamentID, s := range data.Standings {
		if outcome := ComputeTournamentOutcome(s.Outcomes, s.TeamIDs); outcome.WinnerTeamID != nil {
			winners[tournamentID] = *outcome.WinnerTeamID
		}
	}

	cups := make(map[uuid.UUID]int)
	for _, m := range data.Memberships {
		if winner, ok := winners[m.TournamentID]; ok && winner == m.TeamID {
			cups[m.PlayerID]++
		}
	}
	return cups
}

// TournamentResultFor is a player's verdict on one tournament, given the side they were
// drafted onto (nil when they were entered but never drafted).
func TournamentResultFor(s TournamentStandings, teamID *uuid.UUID) sdk.Result {
	outcome := ComputeTournamentOutcome(s.Outcomes, s.TeamIDs)
	switch {
	case !outcome.Finished:
		return sdk.ResultInProgress
	case outcome.WinnerTeamID == nil:
		return sdk.ResultTied
	case teamID != nil && *outcome.WinnerTeamID == *teamID:
		return sdk.ResultWon
	default:
		return sdk.ResultLost
	}
}
