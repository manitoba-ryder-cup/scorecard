package golf

// ComputeMatchProgress computes the hole-by-hole match-play state from the
// recorded scores for a match between teamA and teamB (identified by ID; the
// caller chooses the order). A team's gross score on a hole is the minimum strokes
// recorded for that team on that hole, which uniformly covers singles (the one
// score), fourball (best of two), and one-ball formats (the single team score).
// Only holes scored by both teams contribute, in hole order, and the sequence ends
// at the hole where the match is decided.
//
// The result is pure, color-free state: per-hole TeamScores tagged by ID, the
// LeaderTeamID (nil = all square), and the Lead margin. Rendering it as text is
// the frontend's concern.
func ComputeMatchProgress(scores []Score, teamAID, teamBID int32) []HoleResult {
	a := minStrokesByHole(scores, teamAID)
	b := minStrokesByHole(scores, teamBID)

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
		case sb > sa:
			signed++ // teamA wins the hole (lower score)
		case sa > sb:
			signed-- // teamB wins the hole
		}

		n := i + 1 // scored holes counted so far
		holesRemaining := 18 - n
		lead := abs(signed)
		decided := lead > holesRemaining && n != 18

		var leader *int32
		if signed > 0 {
			id := teamAID
			leader = &id
		} else if signed < 0 {
			id := teamBID
			leader = &id
		}

		result = append(result, HoleResult{
			HoleNumber:     h,
			TeamScores:     []TeamHoleScore{{TeamID: teamAID, Strokes: sa}, {TeamID: teamBID, Strokes: sb}},
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

// minStrokesByHole returns the minimum strokes the given team recorded on each
// hole (best-ball for two players; the single score for singles/one-ball).
func minStrokesByHole(scores []Score, teamID int32) map[int32]int32 {
	m := make(map[int32]int32)
	for _, s := range scores {
		if s.TeamID != teamID {
			continue
		}
		if v, ok := m[s.HoleNumber]; !ok || s.Strokes < v {
			m[s.HoleNumber] = s.Strokes
		}
	}
	return m
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
