package billing

import (
	"testing"

	"github.com/Pototoooo/lorelattice/internal/types"
)

func TestResolveBillingMode(t *testing.T) {
	tests := []struct {
		name          string
		source        types.ModelSource
		key, provider string
		extra         map[string]string
		want          BillingMode
	}{
		{"local", types.ModelSourceLocal, "", "ollama", nil, BillingModeLocal},
		{"remote byok", types.ModelSourceRemote, "sk-user", "siliconflow", nil, BillingModeBYOK},
		{"managed cloud", types.ModelSourceRemote, "", "lorelattice_cloud", nil, BillingModePlatform},
		{"explicit override", types.ModelSourceRemote, "sk-user", "siliconflow", map[string]string{"billing_mode": "included"}, BillingModePlatform},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveBillingMode(tt.source, tt.key, tt.provider, tt.extra); got != tt.want {
				t.Fatalf("ResolveBillingMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNonPlatformModesDoNotCharge(t *testing.T) {
	for _, mode := range []BillingMode{BillingModeBYOK, BillingModeLocal, BillingModeIncluded} {
		if mode.Chargeable() {
			t.Fatalf("mode %q must not debit AI credits", mode)
		}
	}
}

func TestBuiltinRemoteKeyIsPlatformOwned(t *testing.T) {
	model := &types.Model{IsBuiltin: true, Source: types.ModelSourceRemote}
	model.Parameters.APIKey = "platform-secret"
	model.Parameters.ExtraConfig = map[string]string{"region": "test"}
	resolved := ModelBillingConfig(model)
	if got := ResolveBillingMode(model.Source, model.Parameters.APIKey, "siliconflow", resolved); got != BillingModePlatform {
		t.Fatalf("platform key classified as %s", got)
	}
	if _, ok := model.Parameters.ExtraConfig["billing_mode"]; ok {
		t.Fatal("mutated cached model")
	}
	model.IsBuiltin = false
	if got := ResolveBillingMode(model.Source, model.Parameters.APIKey, "siliconflow", ModelBillingConfig(model)); got != BillingModeBYOK {
		t.Fatalf("user key classified as %s", got)
	}
	model.IsBuiltin = true
	model.Source = types.ModelSourceLocal
	if got := ResolveBillingMode(model.Source, "", "ollama", ModelBillingConfig(model)); got != BillingModeLocal {
		t.Fatalf("local built-in classified as %s", got)
	}
}
