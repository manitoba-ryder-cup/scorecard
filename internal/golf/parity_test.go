package golf

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// This is the migration parity test: it replays every real match from the old Python
// (scorecardpy) production database through the Go scoring engine and asserts the Go
// engine derives the exact same match-play state Python computed.
//
// testdata/parity_golden.json is generated from a prod pg_dump (raw gross strokes) +
// the live Python API (its computed per-hole result). Each entry pairs a match's real
// per-player strokes with Python's {red, blue, matchStatus, statusText} per hole plus
// the winner. The fixture carries no PII — only numeric ids, team colors, and strokes.
//
// The APIs deliberately diverged (uuid ids, split schema, auth), so parity is asserted
// at the scoring-engine level, not the HTTP level: raw strokes in, canonical match
// state out, compared to Python's.

type parityFixture struct {
	Matches []parityMatch `json:"matches"`
}

type parityMatch struct {
	ID       int            `json:"id"`
	Format   string         `json:"format"`
	Players  []parityPlayer `json:"players"`
	Scores   []parityScore  `json:"scores"`
	Expected parityExpected `json:"expected"`
}

type parityPlayer struct {
	ID   int    `json:"id"`
	Team string `json:"team"`
}

type parityScore struct {
	P int `json:"p"` // player id
	H int `json:"h"` // hole number
	S int `json:"s"` // strokes
}

type parityExpected struct {
	Finished bool         `json:"finished"`
	Winner   string       `json:"winner"` // "Red" | "Blue" | "Tied"
	Holes    []parityHole `json:"holes"`
}

type parityHole struct {
	Red    int    `json:"red"`    // red team's gross hole score (best ball)
	Blue   int    `json:"blue"`   // blue team's gross hole score
	Status int    `json:"status"` // signed lead, + = Red ahead
	Text   string `json:"text"`   // "AS" | "N UP" | "N & M"
}

func loadParityFixture(t *testing.T) parityFixture {
	t.Helper()
	f, err := os.Open("testdata/parity_golden.json")
	if err != nil {
		t.Fatalf("open parity fixture: %v", err)
	}
	defer func() { _ = f.Close() }()
	var fix parityFixture
	if err := json.NewDecoder(f).Decode(&fix); err != nil {
		t.Fatalf("decode parity fixture: %v", err)
	}
	if len(fix.Matches) == 0 {
		t.Fatal("parity fixture is empty")
	}
	return fix
}

// canonicalHole maps the Go engine's color-free HoleResult back to the Python
// representation (per-team gross score, signed status, rendered status text). This
// mapping IS the parity contract between the two implementations.
func canonicalHole(hr HoleResult, red, blue uuid.UUID) parityHole {
	var r, b int32
	for _, ts := range hr.TeamScores {
		switch ts.TeamID {
		case red:
			r = ts.Strokes
		case blue:
			b = ts.Strokes
		}
	}
	status := 0
	if hr.LeaderTeamID != nil {
		if *hr.LeaderTeamID == red {
			status = hr.Lead
		} else {
			status = -hr.Lead
		}
	}
	var text string
	switch {
	case hr.Decided:
		text = fmt.Sprintf("%d & %d", hr.Lead, hr.HolesRemaining)
	case status == 0:
		text = "AS"
	default:
		text = fmt.Sprintf("%d UP", hr.Lead)
	}
	return parityHole{Red: int(r), Blue: int(b), Status: status, Text: text}
}

func TestParityWithPythonProduction(t *testing.T) {
	fix := loadParityFixture(t)

	var checkedMatches, checkedHoles int
	for _, m := range fix.Matches {
		// Two synthetic team ids; the engine is color-free and works on ids.
		red, blue := uuid.New(), uuid.New()
		team := map[int]uuid.UUID{}
		for _, p := range m.Players {
			switch p.Team {
			case "Red":
				team[p.ID] = red
			case "Blue":
				team[p.ID] = blue
			}
		}

		// One synthetic player id per real player, so best-ball (min of two) is exercised.
		puid := map[int]uuid.UUID{}
		pidFor := func(id int) *uuid.UUID {
			u, ok := puid[id]
			if !ok {
				u = uuid.New()
				puid[id] = u
			}
			v := u
			return &v
		}

		var scores []Score
		for _, s := range m.Scores {
			tid, ok := team[s.P]
			if !ok {
				continue // a non-Red/Blue rostered player (none in the real RvB matches)
			}
			scores = append(scores, Score{
				TeamID:     tid,
				PlayerID:   pidFor(s.P),
				HoleNumber: int32(s.H),
				Strokes:    int32(s.S),
			})
		}

		prog := ComputeMatchProgress(scores, red, blue)
		res := ComputeStoredResult(scores, red, blue)
		exp := m.Expected.Holes

		// The Go engine stops at the hole that decides the match; Python keeps appending
		// holes with the (frozen) closed-out text. So Go's output must equal Python's
		// prefix, and any Python holes beyond it must be that frozen post-decision state.
		if len(prog) > len(exp) {
			t.Errorf("match %d (%s): Go produced %d holes, Python only %d", m.ID, m.Format, len(prog), len(exp))
			continue
		}
		mismatch := false
		for i, hr := range prog {
			got := canonicalHole(hr, red, blue)
			if got != exp[i] {
				t.Errorf("match %d (%s) hole %d: Go %+v != Python %+v", m.ID, m.Format, hr.HoleNumber, got, exp[i])
				mismatch = true
				break
			}
			checkedHoles++
		}
		if mismatch {
			continue
		}
		for i := len(prog); i < len(exp); i++ {
			if !strings.Contains(exp[i].Text, "&") {
				t.Errorf("match %d (%s): Go stopped after %d holes but Python hole %d isn't closed out: %+v",
					m.ID, m.Format, len(prog), i+1, exp[i])
				break
			}
		}

		// Winner + finished from the materialized result.
		if res.Finished != m.Expected.Finished {
			t.Errorf("match %d (%s): finished Go=%v Python=%v", m.ID, m.Format, res.Finished, m.Expected.Finished)
		}
		gotWinner := "Tied"
		if res.LeaderTeamID != nil {
			if *res.LeaderTeamID == red {
				gotWinner = "Red"
			} else {
				gotWinner = "Blue"
			}
		}
		if gotWinner != m.Expected.Winner {
			t.Errorf("match %d (%s): winner Go=%q Python=%q (result %+v)", m.ID, m.Format, gotWinner, m.Expected.Winner, res)
		}
		checkedMatches++
	}

	t.Logf("parity verified: %d matches, %d holes against Python production output", checkedMatches, checkedHoles)
}
