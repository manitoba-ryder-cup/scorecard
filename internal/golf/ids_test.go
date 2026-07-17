package golf

import "github.com/google/uuid"

// Shared, stable UUIDs for the domain tests. Team A / Team B stand in for the two
// sides; the player/match IDs are just distinct non-nil values.
var (
	teamA    = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamB    = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	matchID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	playerA  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	playerB  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	playerA2 = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	playerB2 = uuid.MustParse("55555555-5555-5555-5555-555555555555")

	tournamentID = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	courseID     = uuid.MustParse("77777777-7777-7777-7777-777777777777")
	teeColorID   = uuid.MustParse("88888888-8888-8888-8888-888888888888")
)

func pUUID(u uuid.UUID) *uuid.UUID { return &u }
