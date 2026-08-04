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
		{"explicit override", types.ModelSourceRemote, "sk-user", "siliconflow", map[string]string{"billing_mode": "included"}, BillingModeIncluded},
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
