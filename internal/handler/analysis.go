package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) AnalysisGetIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Analysis.GetIndices)
}

func (h *Handlers) AnalysisGetIndexAnalyzers(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Analysis.GetIndexAnalyzers)
}

func (h *Handlers) AnalysisGetIndexFields(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Analysis.GetIndexFields)
}

func (h *Handlers) AnalysisAnalyzeByField(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Analysis.AnalyzeByField)
}

func (h *Handlers) AnalysisAnalyzeByAnalyzer(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Analysis.AnalyzeByAnalyzer)
}
