package test

import (
	"errors"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// wantsStatus asserts the API refused with the given status and returns the sentence it gave,
// so a caller that also cares what was said can check it without taking the error apart again.
func wantsStatus(t *testing.T, err error, status int) string {
	t.Helper()
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
		t.Fatalf("want a %d APIError, got %v", status, err)
	}
	return apiErr.Message
}
