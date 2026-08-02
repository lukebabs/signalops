package postgres

func valuationLimit(limit int) int {
	if limit <= 0 {
		return 200
	}
	if limit > 5000 {
		return 5000
	}
	return limit
}
