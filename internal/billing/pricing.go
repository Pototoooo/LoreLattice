package billing

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

const defaultPriceVersion = "builtin-v1"

type PriceRule struct {
	Provider     string      `json:"provider"`
	Model        string      `json:"model"`
	Feature      FeatureKey  `json:"feature"`
	BillingMode  BillingMode `json:"billing_mode"`
	UnitPriceUSD float64     `json:"unit_price_usd"`
	Version      string      `json:"version"`
}

type PriceCatalog struct{ rules []PriceRule }

func NewPriceCatalog(raw string) (*PriceCatalog, error) {
	catalog := &PriceCatalog{}
	if strings.TrimSpace(raw) == "" {
		return catalog, nil
	}
	if err := json.Unmarshal([]byte(raw), &catalog.rules); err != nil {
		return nil, fmt.Errorf("parse LORELATTICE_MODEL_PRICING_JSON: %w", err)
	}
	for i := range catalog.rules {
		rule := &catalog.rules[i]
		rule.Provider = strings.ToLower(strings.TrimSpace(rule.Provider))
		rule.Model = strings.TrimSpace(rule.Model)
		if rule.Feature != "" {
			if _, ok := FeatureDefinitions[rule.Feature]; !ok {
				return nil, fmt.Errorf("unknown feature in price rule %d", i)
			}
		}
		if rule.Model != "" {
			if _, err := filepath.Match(rule.Model, ""); err != nil {
				return nil, fmt.Errorf("invalid model pattern in price rule %d", i)
			}
		}
		if rule.Version == "" {
			rule.Version = "env-v1"
		}
		if math.IsNaN(rule.UnitPriceUSD) || math.IsInf(rule.UnitPriceUSD, 0) || rule.UnitPriceUSD < 0 || (rule.BillingMode != "" && !rule.BillingMode.Valid()) {
			return nil, fmt.Errorf("invalid price rule %d", i)
		}
	}
	return catalog, nil
}

func (c *PriceCatalog) Resolve(feature FeatureKey, provider, model string, mode BillingMode) (float64, string) {
	// BYOK, local and included calls are usage telemetry, not LoreLattice spend.
	if !mode.Chargeable() {
		return 0, defaultPriceVersion
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	for _, rule := range c.rules {
		if rule.Feature != "" && rule.Feature != feature {
			continue
		}
		if rule.BillingMode != "" && rule.BillingMode != mode {
			continue
		}
		if rule.Provider != "" && rule.Provider != provider {
			continue
		}
		if rule.Model != "" {
			matched, err := filepath.Match(rule.Model, model)
			if err != nil || !matched {
				continue
			}
		}
		return rule.UnitPriceUSD, rule.Version
	}
	return FeatureDefinitions[feature].UnitPrice, defaultPriceVersion
}
