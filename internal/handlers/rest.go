package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/dao"
	"cerebro/internal/models"
)

func (h *Handlers) RestIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		resp, err := h.Elastic.GetAliases(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		indices, _ := models.AutocompletionIndices(resp.Body)
		data, _ := json.Marshal(map[string]any{
			"indices": json.RawMessage(indices),
			"host":    req.Target.Host.Name,
		})
		return models.NewResponse(resp.Status, data), nil
	})
}

func (h *Handlers) RestRequest(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		method, err := req.Get("method")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		path, err := req.Get("path")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		data := req.GetObjOpt("data")
		// The frontend sends the request body text as a JSON string; unwrap it
		// so ExecuteRequest sees the raw payload (mirrors the Scala
		// executeRequest's JsString handling). Non-string data passes through
		// unchanged (its JSON encoding is the payload).
		if len(data) > 0 && data[0] == '"' {
			var s string
			if json.Unmarshal(data, &s) == nil {
				data = json.RawMessage(s)
			}
		}

		resp, err := h.Elastic.ExecuteRequest(method, path, data, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}

		if resp.Status >= 200 && resp.Status < 300 {
			username := ""
			if req.User != nil {
				username = req.User.Name
			}
			bodyStr := "{}"
			if data != nil {
				var s string
				if json.Unmarshal(data, &s) == nil {
					bodyStr = s
				} else {
					bodyStr = string(data)
				}
			}
			restReq := dao.RestRequest{
				Path:      path,
				Method:    method,
				Body:      bodyStr,
				Username:  username,
				CreatedAt: time.Now(),
			}
			if h.History != nil {
				_, _ = h.History.SaveHistory(restReq, h.HistMax)
			}
		}

		return *resp, nil
	})
}

func (h *Handlers) RestHistory(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		// History is nil when data.path is unset: degrade to an empty list
		// instead of panicking (RestRequest already guards this case).
		if h.History == nil {
			return models.NewResponse(200, json.RawMessage("[]")), nil
		}
		username := ""
		if req.User != nil {
			username = req.User.Name
		}
		entries, err := h.History.AllHistory(username)
		if err != nil {
			return models.ErrorResponse(500, "Error while loading requests history: "+err.Error()), nil
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
		return models.NewResponse(200, body), nil
	})
}
