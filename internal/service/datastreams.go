package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

// DataStreamsService manages data streams and their optional composable
// template bootstrap.
type DataStreamsService struct {
	dataStreams domain.DataStreamAPI
	templates   domain.TemplateAPI
}

func NewDataStreamsService(dataStreams domain.DataStreamAPI, templates domain.TemplateAPI) *DataStreamsService {
	return &DataStreamsService{dataStreams: dataStreams, templates: templates}
}

func (s *DataStreamsService) List(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.dataStreams.GetDataStreams(ctx, req.Target)
	return transform(resp, err, func(body json.RawMessage) (json.RawMessage, error) {
		streams, _, err := domain.ParseDataStreams(body)
		if err != nil {
			return nil, err
		}
		return json.Marshal(streams)
	})
}

func (s *DataStreamsService) Create(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	name, err := req.Get("name")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	if err := domain.ValidateDataStreamName(name); err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}

	createTemplate, _ := req.GetBool("createTemplate")
	if createTemplate {
		templateName := req.GetOpt("templateName")
		if templateName == "" {
			templateName = name + "-ds-template"
		}
		pattern := req.GetOpt("indexPattern")
		if pattern == "" {
			pattern = name + "*"
		}
		priority := intOpt(req, "priority", 200)
		shards := intOpt(req, "numberOfShards", 1)
		replicas := intOpt(req, "numberOfReplicas", 1)

		body, merr := json.Marshal(map[string]any{
			"index_patterns": []string{pattern},
			"data_stream":    map[string]any{},
			"priority":       priority,
			"template": map[string]any{
				"settings": map[string]any{
					"number_of_shards":   shards,
					"number_of_replicas": replicas,
				},
			},
		})
		if merr != nil {
			return domain.ErrorResponse(500, merr.Error()), nil
		}
		resp, terr := s.templates.CreateComposableTemplate(templateName, body, ctx, req.Target)
		if terr != nil {
			return domain.ErrorResponse(502, terr.Error()), nil
		}
		if !resp.IsSuccess() {
			return *resp, nil
		}
	}

	return forward(s.dataStreams.CreateDataStream(name, ctx, req.Target))
}

func (s *DataStreamsService) Delete(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "name", s.dataStreams.DeleteDataStream)
}

func (s *DataStreamsService) Rollover(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "name", s.dataStreams.RolloverDataStream)
}

func intOpt(req domain.CerebroRequest, name string, fallback int) int {
	if v, err := req.GetInt(name); err == nil {
		return v
	}
	return fallback
}
