package asr

import (
	"context"
	"math"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/types"
)

type meterForgeASR struct {
	inner   ASR
	service *billing.Service
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
	reserved := billing.EstimateAudioSeconds(audio)
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureASRSeconds, reserved, billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Operation: "asr.transcribe",
		RequestID: requestID, Estimated: true,
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
	return result, nil
}

func wrapASRMeterForge(a ASR) ASR {
	service := billing.Default()
	if service == nil || !service.Enabled() || a == nil {
		return a
	}
	return &meterForgeASR{inner: a, service: service}
}
