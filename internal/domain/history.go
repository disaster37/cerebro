package domain

import (
	"crypto/md5"
	"fmt"
	"time"
)

// RestRequest is a remembered REST-console request.
type RestRequest struct {
	Path      string
	Method    string
	Body      string
	Username  string
	CreatedAt time.Time
}

func (r RestRequest) MD5() string {
	h := md5.Sum([]byte(r.Path + r.Method + r.Body + r.Username))
	return fmt.Sprintf("%x", h)
}

// RestHistoryRepository persists REST-console history. Implemented by
// internal/repository.SQLiteHistoryRepository.
type RestHistoryRepository interface {
	AllHistory(username string) ([]RestRequest, error)
	SaveHistory(req RestRequest, max int) (string, error)
	ClearHistory(username string) (int64, error)
}
