package golf

import (
	"bytes"
	"slices"

	"github.com/google/uuid"
)

// ComputeMatchProgress computes the hole-by-hole match-play state from the
// recorded scores for a match between teamA and teamB (identified by ID; the
// caller chooses the order). A team's gross score on a hole is the minimum strokes
// recorded for that team on that hole, which uniformly covers singles (the one
// score), fourball (best of two), and one-ball formats (the single team score).
// Only holes scored by both teams contribute, in hole order, and the sequence ends
// at the hole where the match is decided.
//
// The result is pure, color-free state: per-hole TeamScores tagged by ID, the
// LeaderTeamID (nil = all square), and the Lead margin. Decided marks the hole the
// match ended on; whether it ended early is HolesRemaining > 0. Rendering it as text
// is the frontend's concern.
//
// Each TeamScore also carries its per-player breakdown, so a client that recorded a
// score per player can read back more than the side's best ball.
func ComputeMatchProgress(scores []Score, teamAID, teamBID uuid.UUID) []HoleResult {
	a := teamScoresByHole(scores, teamAID)
	b := teamScoresByHole(scores, teamBID)

	// Holes scored by both teams, in order.
	var holes []int32
	for h := int32(1); h <= 18; h++ {
		if _, okA := a[h]; okA {
			if _, okB := b[h]; okB {
				holes = append(holes, h)
			}
		}
	}

	var result []HoleResult
	var signed int // relative to teamA: + teamA ahead, - teamB ahead
	for i, h := range holes {
		sa, sb := a[h], b[h]
		switch {
		case sb.Strokes > sa.Strokes:
			signed++ // teamA wins the hole (lower score)
		case sa.Strokes > sb.Strokes:
			signed-- // teamB wins the hole
		}

		n := i + 1 // scored holes counted so far
		holesRemaining := 18 - n
		lead := abs(signed)
		// Over either because the lead can no longer be caught, or because there is no
		// hole left to play — an all-square 18th is a halved match, which is still over.
		decided := lead > holesRemaining || n == 18

		var leader *uuid.UUID
		if signed > 0 {
			id := teamAID
			leader = &id
		} else if signed < 0 {
			id := teamBID
			leader = &id
		}

		result = append(result, HoleResult{
			HoleNumber:     h,
			TeamScores:     []TeamHoleScore{sa, sb},
			LeaderTeamID:   leader,
			Lead:           lead,
			HolesRemaining: holesRemaining,
			Decided:        decided,
		})

		if decided {
			break
		}
	}
	return result
}

// ComputeStoredResult derives a match's materialized state from the hole-by-hole
// progress. The leader/lead/holes-remaining come from the final scored hole.
func ComputeStoredResult(scores []Score, teamAID, teamBID uuid.UUID) StoredResult {
	progress := ComputeMatchProgress(scores, teamAID, teamBID)
	if len(progress) == 0 {
		return StoredResult{HolesRemaining: 18}
	}

	last := progress[len(progress)-1]
	return StoredResult{
		Finished:       last.Decided,
		LeaderTeamID:   last.LeaderTeamID,
		Lead:           last.Lead,
		HolesRemaining: last.HolesRemaining,
	}
}

// HoleWinner returns the team that won a scored hole (lower gross), or nil if halved.
func HoleWinner(h HoleResult) *uuid.UUID {
	if len(h.TeamScores) != 2 {
		return nil
	}
	switch {
	case h.TeamScores[0].Strokes < h.TeamScores[1].Strokes:
		id := h.TeamScores[0].TeamID
		return &id
	case h.TeamScores[1].Strokes < h.TeamScores[0].Strokes:
		id := h.TeamScores[1].TeamID
		return &id
	default:
		return nil // halved
	}
}

// teamScoresByHole groups the given team's scores by hole: the minimum strokes recorded
// (best-ball for two players; the single score for singles/one-ball), plus the individual
// scores behind it so a client can recover what each player shot. Sorted by player ID
// because the store orders by hole only.
func teamScoresByHole(scores []Score, teamID uuid.UUID) map[int32]TeamHoleScore {
	m := make(map[int32]TeamHoleScore)
	for _, s := range scores {
		if s.TeamID != teamID {
			continue
		}
		t, ok := m[s.HoleNumber]
		if !ok {
			t = TeamHoleScore{TeamID: teamID, Strokes: s.Strokes}
		} else if s.Strokes < t.Strokes {
			t.Strokes = s.Strokes
		}
		if s.PlayerID != nil {
			t.PlayerScores = append(t.PlayerScores, PlayerHoleScore{PlayerID: *s.PlayerID, Strokes: s.Strokes})
		}
		m[s.HoleNumber] = t
	}
	for h, t := range m {
		slices.SortFunc(t.PlayerScores, func(a, b PlayerHoleScore) int {
			return bytes.Compare(a.PlayerID[:], b.PlayerID[:])
		})
		m[h] = t
	}
	return m
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
