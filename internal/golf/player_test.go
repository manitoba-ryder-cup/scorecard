package golf

import (
	"context"
	"errors"
	"testing"
)

type fakePlayerDB struct {
	created *CreatePlayerInput
}

func (f *fakePlayerDB) GetPlayer(ctx context.Context, id int32) (*Player, error) { return nil, nil }
func (f *fakePlayerDB) ListPlayers(ctx context.Context) ([]Player, error)        { return nil, nil }
func (f *fakePlayerDB) CreatePlayer(ctx context.Context, in CreatePlayerInput) (*Player, error) {
	f.created = &in
	return &Player{ID: 42, FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, UserID: in.UserID}, nil
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
	if got.ID != 42 || got.FirstName != "Dustin" || got.LastName != "Johnson" {
		t.Errorf("unexpected player: %+v", got)
	}
	if db.created == nil || db.created.Email == nil || *db.created.Email != "dj@example.com" {
		t.Errorf("input not passed through: %+v", db.created)
	}
}

func TestCreatePlayer_RejectsEmptyNames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first string
		last  string
	}{
		{"empty first", "  ", "Johnson"},
		{"empty last", "Dustin", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &fakePlayerDB{}
			svc := &PlayerService{PlayerDB: db}

			_, err := svc.CreatePlayer(context.Background(), CreatePlayerInput{FirstName: tc.first, LastName: tc.last})
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if db.created != nil {
				t.Error("must not write on validation failure")
			}
		})
	}
}
