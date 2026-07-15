package golf

import "errors"

var (
	// ErrPlayerNotFound is returned when a player cannot be found in a tournament
	ErrPlayerNotFound = errors.New("player not found in tournament")

	// ErrMatchNotStarted is returned when attempting operations on a match that hasn't started
	ErrMatchNotStarted = errors.New("match has not started")

	// ErrMatchNotFinished is returned when attempting operations requiring a finished match
	ErrMatchNotFinished = errors.New("match is not finished")

	// ErrNoParticipants is returned when a match has no participants
	ErrNoParticipants = errors.New("match has no participants")

	// ErrTeamNotFound is returned when a team cannot be found
	ErrTeamNotFound = errors.New("team not found")
)
