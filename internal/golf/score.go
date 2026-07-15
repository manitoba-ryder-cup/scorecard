package golf

// CalculateNetStrokes computes net strokes (handicapped score) by subtracting
// the handicap adjustment from gross strokes.
//
// Algorithm:
//
//	net_strokes = strokes - player.get_hdcp_strokes(hole.hdcp)
//
// This is used in handicapped matches where players receive stroke allowances
// based on their handicap and the difficulty of each hole.
func CalculateNetStrokes(grossStrokes int32, playerHdcp float32, holeHdcp int32) int32 {
	hdcpStrokes := GetHandicapStrokes(playerHdcp, holeHdcp)
	return grossStrokes - hdcpStrokes
}
