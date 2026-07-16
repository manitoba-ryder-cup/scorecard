package sdk

// Team colors. A Ryder Cup tournament has exactly two sides, one of each color;
// the database enforces this with a CHECK and a UNIQUE(tournament_id, color). These
// constants are the single source of truth shared by the domain and the wire layer.
const (
	TeamColorRed  = "Red"
	TeamColorBlue = "Blue"
)

// IsValidTeamColor reports whether color is one of the two allowed team colors.
func IsValidTeamColor(color string) bool {
	return color == TeamColorRed || color == TeamColorBlue
}
