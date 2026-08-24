package service

import (
	"context"
	"encoding/json"
	"time"

	"cerebro/internal/domain"
)

type RestService struct {
	aliases domain.AliasAPI
	raw     domain.RawAPI
	history domain.RestHistoryRepository
	histMax int
}

func NewRestService(aliases domain.AliasAPI, raw domain.RawAPI, history domain.RestHistoryRepository, histMax int) *RestService {
	return &RestService{aliases: aliases, raw: raw, history: history, histMax: histMax}
}

func (s *RestService) Index(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.aliases.GetAliases(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	indices, _ := domain.AutocompletionIndices(resp.Body)
	data, _ := json.Marshal(map[string]any{
		"indices": json.RawMessage(indices),
		"host":    req.Target.Host.Name,
	})
	return domain.NewResponse(resp.Status, data), nil
}

func (s *RestService) Request(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	method, err := req.Get("method")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	path, err := req.Get("path")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	data := req.GetObjOpt("data")
	// The frontend sends the request body text as a JSON string; unwrap it
	// so ExecuteRequest sees the raw payload (mirrors the Scala
	// executeRequest's JsString handling). Non-string data passes through
	// unchanged (its JSON encoding is the payload).
	if len(data) > 0 && data[0] == '"' {
		var str string
		if json.Unmarshal(data, &str) == nil {
			data = json.RawMessage(str)
		}
	}

	resp, err := s.raw.ExecuteRequest(method, path, data, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}

	if resp.IsSuccess() {
		username := ""
		if req.User != nil {
			username = req.User.Name
		}
		bodyStr := "{}"
		if data != nil {
			var str string
			if json.Unmarshal(data, &str) == nil {
				bodyStr = str
			} else {
				bodyStr = string(data)
			}
		}
		restReq := domain.RestRequest{
			Path:      path,
			Method:    method,
			Body:      bodyStr,
			Username:  username,
			CreatedAt: time.Now(),
		}
		if s.history != nil {
			_, _ = s.history.SaveHistory(restReq, s.histMax)
		}
	}

	return *resp, nil
}

func (s *RestService) History(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	// History is nil when data.path is unset: degrade to an empty list
	// instead of panicking (Request already guards this case).
	if s.history == nil {
		return domain.NewResponse(200, json.RawMessage("[]")), nil
	}
	username := ""
	if req.User != nil {
		username = req.User.Name
	}
	entries, err := s.history.AllHistory(username)
	if err != nil {
		return domain.ErrorResponse(500, "Error while loading requests history: "+err.Error()), nil
	}
	var items []map[string]any
	for _, e := range entries {
		items = append(items, map[string]any{
			"path":       e.Path,
			"method":     e.Method,
			"body":       e.Body,
			"created_at": e.CreatedAt.Format("02/01 15:04:05"),
		})
	}
	body, _ := json.Marshal(items)
	return domain.NewResponse(200, body), nil
}
