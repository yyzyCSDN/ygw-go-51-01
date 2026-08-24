package model

// MaxVersion returns the larger of two version numbers.
func MaxVersion(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
