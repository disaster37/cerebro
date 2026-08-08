package dao

import "database/sql"

func Migrate(db *sql.DB) error {
	stmt := `
	CREATE TABLE IF NOT EXISTS rest_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		path TEXT NOT NULL,
		method TEXT NOT NULL,
		body TEXT NOT NULL,
		md5 TEXT NOT NULL,
		created_at INTEGER
	);
	CREATE INDEX IF NOT EXISTS username_idx ON rest_requests(username);
	CREATE UNIQUE INDEX IF NOT EXISTS md5_idx ON rest_requests(md5);
	`
	_, err := db.Exec(stmt)
	return err
}
