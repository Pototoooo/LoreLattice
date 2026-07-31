package rerank

import (
	"context"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/types"
)

type meterForgeReranker struct {
	inner    Reranker
	service  *billing.Service
	provider string
}

func (w *meterForgeReranker) GetModelName() string { return w.inner.GetModelName() }
func (w *meterForgeReranker) GetModelID() string   { return w.inner.GetModelID() }

func (w *meterForgeReranker) Rerank(ctx context.Context, query string, documents []string) ([]RankResult, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "Rerank 调用缺少 workspace 计费上下文",
			HTTPStatus: 503, Feature: billing.FeatureRerankTokens,
		})
	}
	values := make([]string, 0, len(documents)+1)
	values = append(values, query)
	values = append(values, documents...)
	quantity := billing.EstimateTokens(values...)
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureRerankTokens, float64(quantity), billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: "rerank", RequestID: requestID, Estimated: true,
	})
	if err != nil {
		return nil, billing.ToAppError(err)
	}
	result, err := w.inner.Rerank(ctx, query, documents)
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return result, err
	}
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, float64(quantity), true); err != nil {
		return nil, err
	}
	return result, nil
}

func wrapRerankerMeterForge(r Reranker, config *RerankerConfig) Reranker {
	service := billing.Default()
	if service == nil || !service.Enabled() || r == nil {
		return r
	}
	return &meterForgeReranker{inner: r, service: service, provider: config.Provider}
}
