package rest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

func TestToPlayerProfileDTO_CombinesPlayerAndRecord(t *testing.T) {
	id := uuid.New()
	p := golf.Player{ID: id, FirstName: "Dustin", LastName: "Johnson", PhotoPath: "dj.jpg"}
	rec := golf.PlayerRecord{Wins: 3, Losses: 1, Ties: 2}

	got := toPlayerProfileDTO(p, rec)

	// Base player fields are promoted from the embedded Player.
	if got.ID != id || got.FirstName != "Dustin" || got.LastName != "Johnson" || got.PhotoPath != "dj.jpg" {
		t.Errorf("player fields not mapped: %+v", got)
	}
	if got.Record.Wins != 3 || got.Record.Losses != 1 || got.Record.Ties != 2 {
		t.Errorf("record not mapped: %+v", got.Record)
	}
}
