package handlers

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) NavbarIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		resp, err := h.Elastic.ClusterHealth(ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		body := resp.Body
		var m map[string]json.RawMessage
		if body != nil {
			_ = json.Unmarshal(body, &m)
		}
		if m == nil {
			m = map[string]json.RawMessage{}
		}
		if req.User != nil {
			userJSON, _ := json.Marshal(req.User.Name)
			m["username"] = userJSON
		}
		if bi, berr := h.Elastic.BackendInfo(ctx, req.Target); berr == nil {
			t, _ := json.Marshal(bi.Type)
			m["backendType"] = t
			v, _ := json.Marshal(bi.Version)
			m["backendVersion"] = v
		}
		body, _ = json.Marshal(m)
		return domain.NewResponse(resp.Status, body), nil
	})
}
