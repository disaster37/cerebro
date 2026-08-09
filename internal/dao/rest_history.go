package dao

import (
	"time"

	"cerebro/internal/domain"
)

func (d *SQLiteDAO) AllHistory(username string) ([]domain.RestRequest, error) {
	rows, err := d.db.Query(
		"SELECT path, method, body, username, created_at FROM rest_requests WHERE username=? ORDER BY created_at DESC",
		username,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.RestRequest
	for rows.Next() {
		var r domain.RestRequest
		var ts int64
		if err := rows.Scan(&r.Path, &r.Method, &r.Body, &r.Username, &ts); err != nil {
			return nil, err
		}
		r.CreatedAt = time.UnixMilli(ts)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *SQLiteDAO) SaveHistory(req domain.RestRequest, max int) (string, error) {
	md5sum := req.MD5()

	var existingID int
	err := d.db.QueryRow("SELECT id FROM rest_requests WHERE md5=?", md5sum).Scan(&existingID)
	if err == nil {
		_, err = d.db.Exec("UPDATE rest_requests SET created_at=? WHERE id=?", req.CreatedAt.UnixMilli(), existingID)
		if err != nil {
			return "", err
		}
		return md5sum, nil
	}

	_, err = d.db.Exec(
		"INSERT INTO rest_requests (username, path, method, body, md5, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		req.Username, req.Path, req.Method, req.Body, md5sum, req.CreatedAt.UnixMilli(),
	)
	if err != nil {
		return "", err
	}

	_, _ = d.db.Exec(
		"DELETE FROM rest_requests WHERE id IN (SELECT id FROM rest_requests WHERE username=? ORDER BY created_at DESC LIMIT -1 OFFSET ?)",
		req.Username, max,
	)
	return md5sum, nil
}

func (d *SQLiteDAO) ClearHistory(username string) (int64, error) {
	res, err := d.db.Exec("DELETE FROM rest_requests WHERE username=?", username)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
