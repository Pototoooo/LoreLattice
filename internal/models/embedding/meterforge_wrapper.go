package embedding

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/models/resultcache"
	"github.com/Pototoooo/lorelattice/internal/types"
)

var embeddingCache = resultcache.New[[]float32]()

type meterForgeEmbedder struct {
	inner    Embedder
	service  *billing.Service
	provider string
	mode     billing.BillingMode
}

func (w *meterForgeEmbedder) GetModelName() string { return w.inner.GetModelName() }
func (w *meterForgeEmbedder) GetDimensions() int   { return w.inner.GetDimensions() }
func (w *meterForgeEmbedder) GetModelID() string   { return w.inner.GetModelID() }

func (w *meterForgeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	key := w.cacheKey(ctx, text)
	if cached, ok := embeddingCache.Get(key); ok {
		return append([]float32(nil), cached...), nil
	}
	result, err := w.call(ctx, []string{text}, func() ([]float32, error) {
		return w.inner.Embed(ctx, text)
	})
	if err == nil {
		embeddingCache.Set(key, append([]float32(nil), result...))
	}
	return result, err
}

func (w *meterForgeEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "Embedding 调用缺少 workspace 计费上下文",
			HTTPStatus: 503, Feature: billing.FeatureEmbeddingTokens,
		})
	}
	result := make([][]float32, len(texts))
	misses := make([]string, 0, len(texts))
	missIndexes := make([]int, 0, len(texts))
	for i, text := range texts {
		if cached, found := embeddingCache.Get(w.cacheKey(ctx, text)); found {
			result[i] = append([]float32(nil), cached...)
			continue
		}
		misses = append(misses, text)
		missIndexes = append(missIndexes, i)
	}
	if len(misses) == 0 {
		return result, nil
	}
	quantity := billing.EstimateTokens(misses...)
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureEmbeddingTokens, float64(quantity), billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: "embedding.batch", RequestID: requestID, JobID: requestID,
		Category: "document_indexing", Mode: w.mode, Estimated: true,
	})
	if err != nil {
		return nil, billing.ToAppError(err)
	}
	generated, err := w.inner.BatchEmbed(ctx, misses)
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return nil, err
	}
	if len(generated) != len(misses) {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, "provider returned unexpected embedding count")
		return nil, fmt.Errorf("provider returned %d embeddings for %d inputs", len(generated), len(misses))
	}
	for i, vector := range generated {
		index := missIndexes[i]
		result[index] = vector
		embeddingCache.Set(w.cacheKey(ctx, misses[i]), append([]float32(nil), vector...))
	}
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, float64(quantity), true); err != nil {
		return nil, err
	}
	return result, nil
}

func (w *meterForgeEmbedder) cacheKey(ctx context.Context, text string) string {
	tenantID, _ := types.SessionTenantIDFromContext(ctx)
	return resultcache.Key(strconv.FormatUint(tenantID, 10), w.GetModelID(), w.GetModelName(), strconv.Itoa(w.GetDimensions()), text)
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
		Operation: "embedding.embed", RequestID: requestID, JobID: requestID,
		Category: "document_indexing", Mode: w.mode, Estimated: true,
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
	mode := billing.ResolveBillingMode(config.Source, config.APIKey, config.Provider, config.ExtraConfig)
	return &meterForgeEmbedder{inner: e, service: service, provider: config.Provider, mode: mode}
}
