package billing

import (
	"strings"

	"github.com/Pototoooo/lorelattice/internal/types"
)

// BillingMode describes who owns the upstream model bill. It is deliberately
// independent from the provider so one model gateway can host both managed
// and user-supplied credentials.
type BillingMode string

const (
	BillingModePlatform BillingMode = "platform"
	BillingModeBYOK     BillingMode = "byok"
	BillingModeLocal    BillingMode = "local"
	BillingModeIncluded BillingMode = "included"
)

func (m BillingMode) Valid() bool {
	switch m {
	case BillingModePlatform, BillingModeBYOK, BillingModeLocal, BillingModeIncluded:
		return true
	default:
		return false
	}
}

func (m BillingMode) Chargeable() bool { return m == BillingModePlatform }
func (m BillingMode) EnforcesQuota() bool {
	return m == BillingModePlatform
}

// ResolveBillingMode applies an explicit per-model override first, then safe
// product defaults. A remote model carrying a user API key is BYOK; local
// Ollama models only produce local telemetry; LoreLatticeCloud is managed.
func ResolveBillingMode(source types.ModelSource, apiKey, provider string, extra map[string]string) BillingMode {
	if extra != nil {
		if explicit := BillingMode(strings.ToLower(strings.TrimSpace(extra["billing_mode"]))); explicit.Valid() {
			if explicit == BillingModeIncluded {
				return BillingModePlatform
			}
			return explicit
		}
	}
	if source == types.ModelSourceLocal {
		return BillingModeLocal
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "lorelattice_cloud" || provider == "lorelatticecloud" {
		return BillingModePlatform
	}
	if strings.TrimSpace(apiKey) != "" {
		return BillingModeBYOK
	}
	return BillingModePlatform
}

func BusinessCategoryForOperation(operation string) string {
	switch {
	case strings.HasPrefix(operation, "embedding"):
		return "document_indexing"
	case strings.HasPrefix(operation, "vlm"):
		return "image_processing"
	case strings.HasPrefix(operation, "asr"):
		return "audio_transcription"
	default:
		return "chat_agent"
	}
}

// ModelBillingConfig preserves credential ownership when an administrator's
// built-in remote model carries a platform API key. A non-built-in user model
// with an API key remains BYOK. Never mutate the cached model's parameters.
func ModelBillingConfig(model *types.Model) map[string]string {
	if model == nil {
		return nil
	}
	extra := model.Parameters.ExtraConfig
	if !model.IsBuiltin || model.Source == types.ModelSourceLocal {
		return extra
	}
	if BillingMode(strings.ToLower(strings.TrimSpace(extra["billing_mode"]))).Valid() {
		return extra
	}
	resolved := make(map[string]string, len(extra)+1)
	for key, value := range extra {
		resolved[key] = value
	}
	resolved["billing_mode"] = string(BillingModePlatform)
	return resolved
}
