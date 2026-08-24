package service

import (
	"context"
	"encoding/json"
	"fmt"

	"cerebro/internal/domain"
)

type ClusterChangesService struct {
	raw domain.RawAPI
}

func NewClusterChangesService(raw domain.RawAPI) *ClusterChangesService {
	return &ClusterChangesService{raw: raw}
}

func (s *ClusterChangesService) Get(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	// Paths need a leading slash: the ES engine sends them verbatim as the
	// HTTP request target (the Scala client prepended "/" in executeRequest).
	apiPaths := []string{"/_aliases", "/_nodes/transport", "/_cluster/state/blocks"}
	responses := make([]json.RawMessage, len(apiPaths))
	for i, path := range apiPaths {
		resp, err := s.raw.ExecuteRequest("GET", path, nil, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		if !resp.IsSuccess() {
			msg := fmt.Sprintf("Failure for [%s]", path)
			body, _ := json.Marshal(map[string]string{"error": msg})
			return domain.Response{Status: resp.Status, Body: body}, nil
		}
		responses[i] = resp.Body
	}

	var allAliases map[string]json.RawMessage
	_ = json.Unmarshal(responses[0], &allAliases)
	var indices []string
	for idx := range allAliases {
		indices = append(indices, idx)
	}

	var stateObj struct {
		ClusterName json.RawMessage `json:"cluster_name"`
		Blocks      struct {
			Indices map[string]json.RawMessage `json:"indices"`
		} `json:"blocks"`
	}
	_ = json.Unmarshal(responses[2], &stateObj)
	for idx, block := range stateObj.Blocks.Indices {
		var b struct {
			Four any `json:"4"`
		}
		_ = json.Unmarshal(block, &b)
		if b.Four != nil {
			indices = append(indices, idx)
		}
	}

	var nodes []string
	nodesRaw := extractAllNames(responses[1])
	_ = json.Unmarshal(nodesRaw, &nodes)

	body, _ := json.Marshal(map[string]any{
		"indices":      indices,
		"nodes":        nodes,
		"cluster_name": stateObj.ClusterName,
	})
	return domain.NewResponse(200, body), nil
}

func extractAllNames(raw json.RawMessage) json.RawMessage {
	var d any
	_ = json.Unmarshal(raw, &d)
	var result []any
	deepCollectNames(d, &result)
	b, _ := json.Marshal(result)
	return b
}

func deepCollectNames(v any, out *[]any) {
	switch m := v.(type) {
	case map[string]any:
		if sub, ok := m["name"]; ok {
			*out = append(*out, sub)
		}
		for _, sv := range m {
			deepCollectNames(sv, out)
		}
	case []any:
		for _, sv := range m {
			deepCollectNames(sv, out)
		}
	}
}
