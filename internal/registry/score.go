package registry

// ComputeScore validates a pack's quality score by recomputing it from its
// constituent metrics. Packs arrive pre-scored from the registry; this function
// provides a read-only verification.
//
// Scoring algorithm:
//   - Base score: topicCount * 5 (capped at 50 points)
//   - Lines bonus: sum of all topic line counts / 100 (capped at 25 points)
//   - Source bonus: sum of all tier1 source counts * 2 (capped at 25 points)
//   - Total is capped at 100
func ComputeScore(pack PackDetail) int {
	// Base: topic count contribution
	base := pack.TopicCount * 5
	if base > 50 {
		base = 50
	}

	// Lines bonus
	totalLines := 0
	totalTier1 := 0
	for _, t := range pack.Topics {
		totalLines += t.Lines
		totalTier1 += t.Tier1Sources
	}

	linesBonus := totalLines / 100
	if linesBonus > 25 {
		linesBonus = 25
	}

	// Source quality bonus
	sourceBonus := totalTier1 * 2
	if sourceBonus > 25 {
		sourceBonus = 25
	}

	total := base + linesBonus + sourceBonus
	if total > 100 {
		total = 100
	}
	return total
}
