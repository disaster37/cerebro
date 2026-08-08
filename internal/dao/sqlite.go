package dao

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type SQLiteDAO struct {
	db *sql.DB
}

func NewSQLiteDAO(path string) (*SQLiteDAO, error) {
	db, err := sql.Open("sqlite", path+"?cache=shared&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteDAO{db: db}, nil
}

func (d *SQLiteDAO) Close() error {
	return d.db.Close()
}
