package postgres

func riskRewardSnapshotLimit(limit int) int {
	if limit <= 0 {
		return 1_000
	}
	if limit > 10_000 {
		return 10_000
	}
	return limit
}
