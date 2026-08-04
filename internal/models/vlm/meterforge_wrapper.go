package vlm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/models/resultcache"
	"github.com/Pototoooo/lorelattice/internal/types"
)

var vlmCache = resultcache.New[string]()

type meterForgeVLM struct {
	inner    VLM
	service  *billing.Service
	provider string
	mode     billing.BillingMode
}

func (w *meterForgeVLM) GetModelName() string { return w.inner.GetModelName() }
func (w *meterForgeVLM) GetModelID() string   { return w.inner.GetModelID() }

func (w *meterForgeVLM) Predict(ctx context.Context, images [][]byte, prompt string) (string, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return "", billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "VLM 调用缺少 workspace 计费上下文",
			HTTPStatus: 503, Feature: billing.FeatureLLMTokens,
		})
	}
	keyParts := []string{strconv.FormatUint(tenantID, 10), w.GetModelID(), w.GetModelName(), prompt}
	for _, image := range images {
		digest := sha256.Sum256(image)
		keyParts = append(keyParts, hex.EncodeToString(digest[:]))
	}
	cacheKey := resultcache.Key(keyParts...)
	if cached, found := vlmCache.Get(cacheKey); found {
		return cached, nil
	}
	// Image tokenization varies by vendor. Reserve a conservative 256 tokens
	// per image plus the configured default output allowance.
	reserved := billing.EstimateTokens(prompt) + len(images)*256 + w.service.DefaultCompletionReserve()
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureLLMTokens, float64(reserved), billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: "vlm.predict", RequestID: requestID, JobID: requestID,
		Category: "image_processing", Mode: w.mode, Estimated: true,
	})
	if err != nil {
		return "", billing.ToAppError(err)
	}
	result, err := w.inner.Predict(ctx, images, prompt)
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return result, err
	}
	actual := billing.EstimateTokens(prompt, result) + len(images)*256
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, float64(actual), true); err != nil {
		return "", err
	}
	vlmCache.Set(cacheKey, result)
	return result, nil
}

func wrapVLMMeterForge(v VLM, config *Config) VLM {
	service := billing.Default()
	if service == nil || !service.Enabled() || v == nil {
		return v
	}
	extra := make(map[string]string, len(config.Extra))
	for key, value := range config.Extra {
		if text, ok := value.(string); ok {
			extra[key] = text
		}
	}
	mode := billing.ResolveBillingMode(config.Source, config.APIKey, config.Provider, extra)
	return &meterForgeVLM{inner: v, service: service, provider: config.Provider, mode: mode}
}
