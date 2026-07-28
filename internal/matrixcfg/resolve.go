package matrixcfg

// IntOr returns *v if non-nil, otherwise fallback.
// Use this to implement the priority chain: CLI flag > config > default.
func IntOr(v *int, fallback int) int {
	if v != nil {
		return *v
	}
	return fallback
}

// Float64Or returns *v if non-nil, otherwise fallback.
func Float64Or(v *float64, fallback float64) float64 {
	if v != nil {
		return *v
	}
	return fallback
}

// StringOr returns *v if non-nil, otherwise fallback.
func StringOr(v *string, fallback string) string {
	if v != nil {
		return *v
	}
	return fallback
}

// BoolOr returns *v if non-nil, otherwise fallback.
// Note: false is a valid value — only nil means "not set".
func BoolOr(v *bool, fallback bool) bool {
	if v != nil {
		return *v
	}
	return fallback
}
