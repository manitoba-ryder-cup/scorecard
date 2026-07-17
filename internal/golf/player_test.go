package golf

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakePlayerDB struct {
	created *CreatePlayerInput
}

func (f *fakePlayerDB) GetPlayer(ctx context.Context, id uuid.UUID) (*Player, error) { return nil, nil }
func (f *fakePlayerDB) ListPlayers(ctx context.Context) ([]Player, error)            { return nil, nil }
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
