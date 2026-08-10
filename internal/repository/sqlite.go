package repository

import (
	"database/sql"

	"cerebro/internal/domain"

	_ "modernc.org/sqlite"
)

type SQLiteHistoryRepository struct {
	db *sql.DB
}

var _ domain.RestHistoryRepository = (*SQLiteHistoryRepository)(nil)

func NewSQLiteHistoryRepository(path string) (*SQLiteHistoryRepository, error) {
	db, err := sql.Open("sqlite", path+"?cache=shared&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteHistoryRepository{db: db}, nil
}

func (d *SQLiteHistoryRepository) Close() error {
	return d.db.Close()
}
