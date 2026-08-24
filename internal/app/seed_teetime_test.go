package app

import (
	"testing"
	"time"
)

func TestParseTeeTime(t *testing.T) {
	// 08:00 in Manitoba is 13:00Z; stored as 08:00Z it reads back as a 3am tee time.
	t.Run("a bare wall clock is read in the tournament's zone", func(t *testing.T) {
		got, err := parseTeeTime("2026-09-18T08:00", "America/Winnipeg")
		if err != nil {
			t.Fatalf("parseTeeTime: %v", err)
		}
		if want := "2026-09-18T13:00:00Z"; got.UTC().Format(time.RFC3339) != want {
			t.Errorf("got %s, want %s", got.UTC().Format(time.RFC3339), want)
		}
	})

	t.Run("seconds are optional", func(t *testing.T) {
		with, err1 := parseTeeTime("2026-09-18T08:00:00", "America/Winnipeg")
		without, err2 := parseTeeTime("2026-09-18T08:00", "America/Winnipeg")
		if err1 != nil || err2 != nil || !with.Equal(without) {
			t.Errorf("want the same instant, got %v/%v (%v, %v)", with, without, err1, err2)
		}
	})

	t.Run("an away event uses its own zone", func(t *testing.T) {
		got, err := parseTeeTime("2026-09-18T08:00", "America/Phoenix")
		if err != nil {
			t.Fatalf("parseTeeTime: %v", err)
		}
		if want := "2026-09-18T15:00:00Z"; got.UTC().Format(time.RFC3339) != want {
			t.Errorf("got %s, want %s", got.UTC().Format(time.RFC3339), want)
		}
	})

	t.Run("an explicit offset is trusted as given", func(t *testing.T) {
		got, err := parseTeeTime("2026-09-18T13:00:00Z", "America/Winnipeg")
		if err != nil {
			t.Fatalf("parseTeeTime: %v", err)
		}
		if want := "2026-09-18T13:00:00Z"; got.UTC().Format(time.RFC3339) != want {
			t.Errorf("got %s, want %s", got.UTC().Format(time.RFC3339), want)
		}
	})

	t.Run("nonsense is rejected", func(t *testing.T) {
		if _, err := parseTeeTime("half past eight", "America/Winnipeg"); err == nil {
			t.Error("want an error for an unparseable tee time")
		}
	})
}
