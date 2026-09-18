package asr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/models/resultcache"
	"github.com/Pototoooo/lorelattice/internal/types"
)

var asrCache = resultcache.New[*TranscriptionResult]()

type meterForgeASR struct {
	inner    ASR
	service  *billing.Service
	provider string
	mode     billing.BillingMode
}

func (w *meterForgeASR) GetModelName() string { return w.inner.GetModelName() }
func (w *meterForgeASR) GetModelID() string   { return w.inner.GetModelID() }

func (w *meterForgeASR) Transcribe(ctx context.Context, audio []byte, fileName string) (*TranscriptionResult, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "ASR 调用缺少 workspace 计费上下文",
			HTTPStatus: 503, Feature: billing.FeatureASRSeconds,
		})
	}
	digest := sha256.Sum256(audio)
	cacheKey := resultcache.Key(strconv.FormatUint(tenantID, 10), w.GetModelID(), w.GetModelName(), fileName, hex.EncodeToString(digest[:]))
	if cached, found := asrCache.Get(cacheKey); found {
		return cloneTranscription(cached), nil
	}
	reserved := billing.EstimateAudioSeconds(audio)
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureASRSeconds, reserved, billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: "asr.transcribe", RequestID: requestID, JobID: requestID,
		Category: "audio_transcription", Mode: w.mode, Estimated: true,
	})
	if err != nil {
		return nil, billing.ToAppError(err)
	}
	result, err := w.inner.Transcribe(ctx, audio, fileName)
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return result, err
	}
	actual, estimated := reserved, true
	if result != nil && len(result.Segments) > 0 {
		actual = math.Ceil(result.Segments[len(result.Segments)-1].End)
		estimated = false
	}
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, actual, estimated); err != nil {
		return nil, err
	}
	asrCache.Set(cacheKey, cloneTranscription(result))
	return result, nil
}

func cloneTranscription(in *TranscriptionResult) *TranscriptionResult {
	if in == nil {
		return nil
	}
	copy := *in
	copy.Segments = append([]Segment(nil), in.Segments...)
	return &copy
}

func wrapASRMeterForge(a ASR, config *Config) ASR {
	service := billing.Default()
	if service == nil || !service.Enabled() || a == nil || config == nil {
		return a
	}
	mode := billing.ResolveBillingMode(config.Source, config.APIKey, config.Provider, config.ExtraConfig)
	return &meterForgeASR{inner: a, service: service, provider: config.Provider, mode: mode}
}
