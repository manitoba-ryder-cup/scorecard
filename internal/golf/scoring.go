package golf

// ComputeMatchProgress computes the hole-by-hole match-play state from the
// recorded scores. A team's gross score on a hole is the minimum strokes recorded
// for that team on that hole, which uniformly covers singles (the one score),
// fourball (best of two), and one-ball formats (the single team score). Only holes
// scored by both teams contribute, in hole order, and the sequence ends at the
// hole where the match is decided (closed out).
//
// The result is pure state — Lead (+ Red, - Blue, 0 square) and Decided. Rendering
// it as text ("AS"/"2 UP"/"3 & 2") is a separate concern (FormatHoleStatus).
func ComputeMatchProgress(scores []Score, redTeamID, blueTeamID int32) []HoleResult {
	red := minStrokesByHole(scores, redTeamID)
	blue := minStrokesByHole(scores, blueTeamID)

	// Holes scored by both teams, in order.
	var holes []int32
	for h := int32(1); h <= 18; h++ {
		if _, okR := red[h]; okR {
			if _, okB := blue[h]; okB {
				holes = append(holes, h)
			}
		}
	}

	var result []HoleResult
	var lead int
	for i, h := range holes {
		r, b := red[h], blue[h]
		switch {
		case b > r:
			lead++ // Red wins the hole (lower score)
		case r > b:
			lead-- // Blue wins the hole
		}

		n := i + 1 // scored holes counted so far
		holesRemaining := 18 - n
		// Decided when the lead can no longer be caught (and not on the 18th).
		decided := abs(lead) > holesRemaining && n != 18

		result = append(result, HoleResult{
			HoleNumber:     h,
			RedStrokes:     r,
			BlueStrokes:    b,
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
