package repo

// btoi converts a bool to an int (1 for true, 0 for false).
// This is needed because PostgreSQL strict typing rejects "true"/"false"
// strings for INTEGER columns, whereas SQLite is lenient.
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
