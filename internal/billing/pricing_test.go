package billing

import "testing"

func TestPriceCatalogPrefersMatchingModelRule(t *testing.T) {
	catalog, err := NewPriceCatalog(`[
      {"provider":"siliconflow","model":"Qwen/*","feature":"lorelattice_llm_tokens","billing_mode":"platform","unit_price_usd":0.00002,"version":"sf-2026-08"}
    ]`)
	if err != nil {
		t.Fatal(err)
	}
	price, version := catalog.Resolve(FeatureLLMTokens, "SiliconFlow", "Qwen/Qwen3-32B", BillingModePlatform)
	if price != 0.00002 || version != "sf-2026-08" {
		t.Fatalf("got price=%v version=%q", price, version)
	}
}

func TestPriceCatalogBYOKAlwaysZero(t *testing.T) {
	catalog, err := NewPriceCatalog(`[{"provider":"siliconflow","unit_price_usd":9,"version":"bad-for-byok"}]`)
	if err != nil {
		t.Fatal(err)
	}
	price, _ := catalog.Resolve(FeatureLLMTokens, "siliconflow", "Qwen/Qwen3", BillingModeBYOK)
	if price != 0 {
		t.Fatalf("BYOK price = %v, want 0", price)
	}
}
