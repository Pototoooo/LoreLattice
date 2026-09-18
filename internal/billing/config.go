package billing

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type FeatureKey string

const (
	FeatureLLMTokens       FeatureKey = "lorelattice_llm_tokens"
	FeatureEmbeddingTokens FeatureKey = "lorelattice_embedding_tokens"
	FeatureRerankTokens    FeatureKey = "lorelattice_rerank_tokens"
	FeatureASRSeconds      FeatureKey = "lorelattice_asr_seconds"

	PlanTrial = "lorelattice_trial"
	PlanPro   = "lorelattice_pro"
)

type FeatureDefinition struct {
	Key        FeatureKey
	Name       string
	MeterSlug  string
	EventType  string
	Unit       string
	UnitPrice  float64
	ValueField string
}

var FeatureDefinitions = map[FeatureKey]FeatureDefinition{
	FeatureLLMTokens: {
		Key: FeatureLLMTokens, Name: "LoreLattice LLM / VLM Tokens",
		MeterSlug: "lorelattice_llm_tokens_total", EventType: "lorelattice.llm.tokens",
		Unit: "token", UnitPrice: 0.0001, ValueField: "$.quantity",
	},
	FeatureEmbeddingTokens: {
		Key: FeatureEmbeddingTokens, Name: "LoreLattice Embedding Tokens",
		MeterSlug: "lorelattice_embedding_tokens_total", EventType: "lorelattice.embedding.tokens",
		Unit: "token", UnitPrice: 0.00001, ValueField: "$.quantity",
	},
	FeatureRerankTokens: {
		Key: FeatureRerankTokens, Name: "LoreLattice Rerank Tokens",
		MeterSlug: "lorelattice_rerank_tokens_total", EventType: "lorelattice.rerank.tokens",
		Unit: "token", UnitPrice: 0.00002, ValueField: "$.quantity",
	},
	FeatureASRSeconds: {
		Key: FeatureASRSeconds, Name: "LoreLattice ASR Seconds",
		MeterSlug: "lorelattice_asr_seconds_total", EventType: "lorelattice.asr.seconds",
		Unit: "second", UnitPrice: 0.0001, ValueField: "$.quantity",
	},
}

type Config struct {
	Enabled                  bool
	MeterForgeEnabled        bool
	WelcomeCreditUSD         float64
	CNYPerUSD                float64
	ReservationTTL           time.Duration
	BaseURL                  string
	APIKey                   string
	Source                   string
	SubjectPrefix            string
	Timeout                  time.Duration
	OutboxInterval           time.Duration
	DefaultCompletionReserve int
	ModelPricingJSON         string
}

func LoadConfigFromEnv() Config {
	cfg := Config{
		Enabled:                  envBool("BILLING_ENABLED"),
		MeterForgeEnabled:        envBool("METERFORGE_ENABLED"),
		WelcomeCreditUSD:         envFloat("BILLING_WELCOME_CREDIT_USD", 0),
		CNYPerUSD:                envFloat("BILLING_CNY_PER_USD", 7),
		ReservationTTL:           envDuration("BILLING_RESERVATION_TTL", 24*time.Hour),
		BaseURL:                  envString("METERFORGE_BASE_URL", "http://127.0.0.1:48888"),
		APIKey:                   strings.TrimSpace(os.Getenv("METERFORGE_API_KEY")),
		Source:                   envString("METERFORGE_SOURCE", "lorelattice"),
		SubjectPrefix:            envString("METERFORGE_SUBJECT_PREFIX", "tenant:"),
		Timeout:                  envDuration("METERFORGE_TIMEOUT", 3*time.Second),
		OutboxInterval:           envDuration("METERFORGE_OUTBOX_INTERVAL", 5*time.Second),
		DefaultCompletionReserve: envInt("METERFORGE_DEFAULT_COMPLETION_RESERVE", 256),
		ModelPricingJSON:         strings.TrimSpace(os.Getenv("LORELATTICE_MODEL_PRICING_JSON")),
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg
}

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envDuration(name string, fallback time.Duration) time.Duration {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		if value, err := time.ParseDuration(raw); err == nil && value > 0 {
			return value
		}
	}
	return fallback
}

func envInt(name string, fallback int) int {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return value
		}
	}
	return fallback
}

func envFloat(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if value, err := strconv.ParseFloat(raw, 64); err == nil && value >= 0 && value <= 1000000 {
		return value
	}
	return fallback
}
