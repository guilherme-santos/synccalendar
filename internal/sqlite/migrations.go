package sqlite

func (s Storage) RunMigrations() error {
	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return err
		}
	}
	return nil
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id VARCHAR NOT NULL PRIMARY KEY,
		auth TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS calendars (
		account_id VARCHAR NOT NULL,
		name VARCHAR NOT NULL,
		provider_id VARCHAR NOT NULL,
		last_sync VARCHAR NOT NULL DEFAULT "",
		dst_calendar_id VARCHAR NULL DEFAULT NULL,
		PRIMARY KEY (account_id, name),
		FOREIGN KEY (account_id) REFERENCES accounts (id)
	)`,
	`CREATE TABLE IF NOT EXISTS events (
		calendar_id VARCHAR NOT NULL,
		provider_id VARCHAR NOT NULL,
		src_provider_id VARCHAR NOT NULL,
		PRIMARY KEY (calendar_id, provider_id)
	)`,
}
