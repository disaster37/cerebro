// Package service holds the feature services: the business logic extracted
// from the HTTP handlers. Each service depends only on domain types and on
// the narrow domain capability interfaces it actually uses (ISP). Every
// method wired through handler.process has the handler.Processor signature
// func(ctx context.Context, req domain.CerebroRequest) (domain.Response,
// error), so handlers delegate to services with a plain method value.
package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

// forward converts a client call result into a service result: transport
// errors become a {502, error} envelope; successful responses are unwrapped
// and returned verbatim (the services are pass-through for raw ES/OS data).
func forward(resp *domain.Response, err error) (domain.Response, error) {
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

// transform applies fn to the body of a successful client response and
// returns the transformed body with the original status: transport errors
// become a {502, error} envelope, transformation errors a {500, error}
// envelope.
func transform(resp *domain.Response, err error, fn func(json.RawMessage) (json.RawMessage, error)) (domain.Response, error) {
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := fn(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

// forwardOp extracts the required param from req, calls op with it, and
// forwards the client response. It captures the "one required param + one
// client call" shape shared by many service methods.
func forwardOp(ctx context.Context, req domain.CerebroRequest, param string,
	op func(v string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error),
) (domain.Response, error) {
	v, err := req.Get(param)
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	return forward(op(v, ctx, req.Target))
}
