package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

type NavbarService struct {
	cluster domain.ClusterAPI
}

func NewNavbarService(cluster domain.ClusterAPI) *NavbarService {
	return &NavbarService{cluster: cluster}
}

func (s *NavbarService) Index(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.cluster.ClusterHealth(ctx, req.Target)
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
	if bi, berr := s.cluster.BackendInfo(ctx, req.Target); berr == nil {
		t, _ := json.Marshal(bi.Type)
		m["backendType"] = t
		v, _ := json.Marshal(bi.Version)
		m["backendVersion"] = v
	}
	body, _ = json.Marshal(m)
	return domain.NewResponse(resp.Status, body), nil
}
