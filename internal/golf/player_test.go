package golf

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakePlayerDB struct {
	created   *CreatePlayerInput
	history   []PlayerTournamentHistory
	byFormat  []FormatRecord
	teammates []PairRecord
	opponents []PairRecord
	lastHole  PlayerRecord
	early     PlayerRecord
}

func (f *fakePlayerDB) GetPlayer(ctx context.Context, id uuid.UUID) (*Player, error) { return nil, nil }
func (f *fakePlayerDB) ListPlayers(ctx context.Context) ([]Player, error)            { return nil, nil }
func (f *fakePlayerDB) ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]PlayerTournamentHistory, error) {
	return f.history, nil
}
func (f *fakePlayerDB) PlayerStatsRows(ctx context.Context, playerID uuid.UUID) (*PlayerStatsRows, error) {
	return &PlayerStatsRows{
		ByFormat:     f.byFormat,
		Teammates:    f.teammates,
		Opponents:    f.opponents,
		LastHole:     f.lastHole,
		DecidedEarly: f.early,
		History:      f.history,
	}, nil
}
func (f *fakePlayerDB) CreatePlayer(ctx context.Context, in CreatePlayerInput) (*Player, error) {
	f.created = &in
	return &Player{ID: playerA, FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, UserID: in.UserID}, nil
}

func TestCreatePlayer_Valid(t *testing.T) {
	db := &fakePlayerDB{}
	svc := &PlayerService{PlayerDB: db}

	email := "dj@example.com"
	got, err := svc.CreatePlayer(context.Background(), CreatePlayerInput{
		FirstName: "Dustin", LastName: "Johnson", Email: &email,
	})
	if err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	if got.ID != playerA || got.FirstName != "Dustin" || got.LastName != "Johnson" {
		t.Errorf("unexpected player: %+v", got)
	}
	if db.created == nil || db.created.Email == nil || *db.created.Email != "dj@example.com" {
		t.Errorf("input not passed through: %+v", db.created)
	}
}

func TestPlayerStats_PointsAreHalvesAndWins(t *testing.T) {
	// A half is worth half a point, which is the whole reason points and W-L-T are
	// different numbers: 5-3-2 is six points, not five.
	db := &fakePlayerDB{
		byFormat: []FormatRecord{
			{FormatName: "Singles", Record: PlayerRecord{Wins: 3, Losses: 2, Ties: 1}},
			{FormatName: "Fourball", Record: PlayerRecord{Wins: 2, Losses: 1, Ties: 1}},
		},
		history: make([]PlayerTournamentHistory, 4),
	}
	svc := &PlayerService{PlayerDB: db}

	got, err := svc.PlayerStats(context.Background(), playerA)
	if err != nil {
		t.Fatalf("PlayerStats: %v", err)
	}
	if got.Points != 6 {
		t.Errorf("points: want 6 (5 wins + 2 halves), got %v", got.Points)
	}
	if got.CupsPlayed != 4 {
		t.Errorf("cups played: want 4, got %d", got.CupsPlayed)
	}
}

func TestPlayerStats_PointsMatchTheRowsBeneathThem(t *testing.T) {
	// Summed from the same split the page renders, so a total can never disagree with the
	// rows under it.
	db := &fakePlayerDB{byFormat: []FormatRecord{
		{FormatName: "Singles", Record: PlayerRecord{Wins: 1, Losses: 0, Ties: 1}},
	}}
	svc := &PlayerService{PlayerDB: db}

	got, _ := svc.PlayerStats(context.Background(), playerA)
	var fromRows float64
	for _, f := range got.ByFormat {
		fromRows += PointsFor(f.Record)
	}
	if got.Points != fromRows {
		t.Errorf("total %v disagrees with the rows (%v)", got.Points, fromRows)
	}
}

func TestPlayerStats_ClosenessSplitAccountsForEveryMatch(t *testing.T) {
	// The two buckets partition the finished matches, so together they have to come to
	// the same W-L-T as the format split. A half can only be in the last-hole bucket: a
	// match is only halved by playing the 18th.
	db := &fakePlayerDB{
		byFormat: []FormatRecord{
			{FormatName: "Singles", Record: PlayerRecord{Wins: 5, Losses: 4, Ties: 1}},
		},
		lastHole: PlayerRecord{Wins: 2, Losses: 1, Ties: 1},
		early:    PlayerRecord{Wins: 3, Losses: 3, Ties: 0},
	}
	svc := &PlayerService{PlayerDB: db}

	got, err := svc.PlayerStats(context.Background(), playerA)
	if err != nil {
		t.Fatalf("PlayerStats: %v", err)
	}
	total := PlayerRecord{
		Wins:   got.LastHole.Wins + got.DecidedEarly.Wins,
		Losses: got.LastHole.Losses + got.DecidedEarly.Losses,
		Ties:   got.LastHole.Ties + got.DecidedEarly.Ties,
	}
	if total != got.ByFormat[0].Record {
		t.Errorf("closeness split %+v does not account for the format record %+v", total, got.ByFormat[0].Record)
	}
}

func TestCreatePlayer_TrimsNames(t *testing.T) {
	// The boundary trims before checking a name isn't blank, then stores what was typed —
	// so "Graydon " passed validation and was saved with the space, printing as a double
	// space wherever a full name is assembled.
	db := &fakePlayerDB{}
	svc := &PlayerService{PlayerDB: db}

	if _, err := svc.CreatePlayer(context.Background(), CreatePlayerInput{FirstName: "Graydon ", LastName: " Cramer "}); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	if db.created.FirstName != "Graydon" || db.created.LastName != "Cramer" {
		t.Errorf("want names trimmed, stored %q / %q", db.created.FirstName, db.created.LastName)
	}
}
