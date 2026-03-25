package db

// GetLifetimeTokens returns the accumulated lifetime token counts.
func GetLifetimeTokens(db *DB) (totalInput, totalOutput int64, err error) {
	err = db.QueryRow(
		"SELECT total_input, total_output FROM lifetime_tokens WHERE id = 1",
	).Scan(&totalInput, &totalOutput)
	return
}

// AddLifetimeTokens adds the given amounts to the lifetime token counters.
func AddLifetimeTokens(db *DB, inputTokens, outputTokens int64) error {
	_, err := db.Exec(
		`UPDATE lifetime_tokens
		 SET total_input = total_input + ?, total_output = total_output + ?
		 WHERE id = 1`,
		inputTokens, outputTokens)
	return err
}
