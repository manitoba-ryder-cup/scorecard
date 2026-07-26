package test

import (
	"strings"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

// A value longer than its column trips Postgres 22001 (string_data_right_truncation).
// That is the caller sending something too long, so it must read as 400 rather than
// surfacing as a server fault.
func TestOverlongFieldsAreRejectedAsBadRequest(t *testing.T) {
	t.Parallel()
	token := freshToken(t)

	tests := []struct {
		name   string
		path   string
		body   string
		column string
	}{
		{"player first name", "/v1/players",
			`{"first_name":"` + strings.Repeat("a", 40) + `","last_name":"Smith"}`, "first_name VARCHAR(32)"},
		{"player last name", "/v1/players",
			`{"first_name":"Bob","last_name":"` + strings.Repeat("a", 40) + `"}`, "last_name VARCHAR(32)"},
		{"tee color", "/v1/tee-colors",
			`{"color":"` + strings.Repeat("a", 40) + `"}`, "color VARCHAR(32)"},
		{"course name", "/v1/courses",
			`{"name":"` + strings.Repeat("a", 300) + `"}`, "name VARCHAR(255)"},
		{"tournament name", "/v1/tournaments",
			`{"name":"` + strings.Repeat("a", 300) + `","start_date":"2026-08-01","end_date":"2026-08-03","location":"Winnipeg"}`, "name VARCHAR(255)"},
		{"tournament location", "/v1/tournaments",
			`{"name":"Cup","start_date":"2026-08-01","end_date":"2026-08-03","location":"` + strings.Repeat("a", 300) + `"}`, "location VARCHAR(255)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body := request.Raw(t, "POST", tc.path, tc.body, token)
			if status != 400 {
				t.Errorf("POST %s with an over-long %s: status %d, want 400 (body: %s)",
					tc.path, tc.column, status, strings.TrimSpace(body))
			}
		})
	}
}
