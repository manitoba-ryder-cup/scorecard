package sdk

import (
	"context"
	"strings"
	"testing"
)

// TestClientValidatesBeforeSending confirms the client runs request validation before
// any network call: the base URL is unreachable, so a transport error would prove it
// tried to connect. Instead it must fail on validation.
func TestClientValidatesBeforeSending(t *testing.T) {
	c := NewClient("http://127.0.0.1:1") // nothing listens here

	_, err := c.CreatePlayer(context.Background(), CreatePlayerRequest{FirstName: "", LastName: "Johnson"})
	if err == nil {
		t.Fatal("want a validation error before sending")
	}
	if !strings.Contains(err.Error(), "first_name is required") {
		t.Fatalf("want client-side validation error, got %v", err)
	}
}
