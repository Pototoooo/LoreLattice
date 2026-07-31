package embedding

import (
	"context"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/types"
)

type meterForgeEmbedder struct {
	inner    Embedder
	service  *billing.Service
	provider string
}

func (w *meterForgeEmbedder) GetModelName() string { return w.inner.GetModelName() }
func (w *meterForgeEmbedder) GetDimensions() int   { return w.inner.GetDimensions() }
func (w *meterForgeEmbedder) GetModelID() string   { return w.inner.GetModelID() }

func (w *meterForgeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return w.call(ctx, []string{text}, func() ([]float32, error) {
		return w.inner.Embed(ctx, text)
	})
}

func (w *meterForgeEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "Embedding 调用缺少 workspace 计费上下文",
			HTTPStatus: 503, Feature: billing.FeatureEmbeddingTokens,
		})
	}
	quantity := billing.EstimateTokens(texts...)
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureEmbeddingTokens, float64(quantity), billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: "embedding.batch", RequestID: requestID, Estimated: true,
	})
	if err != nil {
		return nil, billing.ToAppError(err)
	}
	result, err := w.inner.BatchEmbed(ctx, texts)
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return result, err
	}
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, float64(quantity), true); err != nil {
		return nil, err
	}
	return result, nil
}

func (w *meterForgeEmbedder) BatchEmbedWithPool(ctx context.Context, _ Embedder, texts []string) ([][]float32, error) {
	// Pass the billing wrapper as the model so every provider sub-batch still
	// traverses the reservation boundary exactly once.
	return w.inner.BatchEmbedWithPool(ctx, w, texts)
}

func (w *meterForgeEmbedder) call(ctx context.Context, texts []string, invoke func() ([]float32, error)) ([]float32, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "Embedding 调用缺少 workspace 计费上下文",
			HTTPStatus: 503, Feature: billing.FeatureEmbeddingTokens,
		})
	}
	quantity := billing.EstimateTokens(texts...)
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureEmbeddingTokens, float64(quantity), billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: "embedding.embed", RequestID: requestID, Estimated: true,
	})
	if err != nil {
		return nil, billing.ToAppError(err)
	}
	result, err := invoke()
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return result, err
	}
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, float64(quantity), true); err != nil {
		return nil, err
	}
	return result, nil
}

func wrapEmbeddingMeterForge(e Embedder, config Config) Embedder {
	service := billing.Default()
	if service == nil || !service.Enabled() || e == nil {
		return e
	}
	return &meterForgeEmbedder{inner: e, service: service, provider: config.Provider}
}
