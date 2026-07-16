package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

func TestRespondDomainError_StatusMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not found", golf.ErrNotFound, http.StatusNotFound},
		{"invalid input", golf.ErrInvalidInput, http.StatusBadRequest},
		{"conflict", golf.ErrConflict, http.StatusConflict},
		// The sentinel is matched through wrapping, as repos/services wrap it.
		{"wrapped not found", fmt.Errorf("getting player 5: %w", golf.ErrNotFound), http.StatusNotFound},
		{"unknown error", errors.New("connection reset"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondDomainError(rec, "message", tc.err)
			if rec.Code != tc.want {
				t.Fatalf("want status %d, got %d", tc.want, rec.Code)
			}
		})
	}
}
