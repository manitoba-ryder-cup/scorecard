package golf

import "testing"

func TestFormatHoleStatus(t *testing.T) {
	cases := []struct {
		name string
		h    HoleResult
		want string
	}{
		{"all square", HoleResult{Lead: 0}, "AS"},
		{"two up", HoleResult{Lead: 2}, "2 UP"},
		{"closed out", HoleResult{Lead: 3, HolesRemaining: 2, Decided: true}, "3 & 2"},
	}
	for _, c := range cases {
		if got := FormatHoleStatus(c.h); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
