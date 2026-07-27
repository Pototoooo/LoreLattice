package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Pototoooo/lorelattice/internal/models/provider"
	"github.com/Pototoooo/lorelattice/internal/models/utils"
	"github.com/google/uuid"
)

const loreLatticeCloudEmbedPath = "/api/v1/embeddings"

// LoreLatticeCloudEmbedder 实现 embedding.Embedder 接口，对接 LoreLatticeCloud /api/v1/embeddings
type LoreLatticeCloudEmbedder struct {
	modelName                 string
	remoteModelName           string
	modelID                   string
	appID                     string
	apiKey                    string
	baseURL                   string
	dimensions                int
	supportsDimensionOverride bool
	client                    *http.Client
	EmbedderPooler
}

// NewLoreLatticeCloudEmbedder 构造 LoreLatticeCloudEmbedder
func NewLoreLatticeCloudEmbedder(config Config) (*LoreLatticeCloudEmbedder, error) {
	if config.AppID == "" {
		return nil, fmt.Errorf("LoreLatticeCloud embedder: AppID is required")
	}
	if config.AppSecret == "" {
		return nil, fmt.Errorf("LoreLatticeCloud embedder: AppSecret is required")
	}
	remoteModelName := ""
	if config.ExtraConfig != nil {
		remoteModelName = strings.TrimSpace(config.ExtraConfig["remote_model_name"])
	}
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		baseURL = provider.LoreLatticeCloudBaseURL
	}
	if err := validateEmbeddingBaseURL(baseURL); err != nil {
		return nil, err
	}
	return &LoreLatticeCloudEmbedder{
		modelName:                 config.ModelName,
		remoteModelName:           remoteModelName,
		modelID:                   config.ModelID,
		appID:                     config.AppID,
		apiKey:                    config.AppSecret,
		baseURL:                   baseURL,
		dimensions:                config.Dimensions,
		supportsDimensionOverride: config.SupportsDimensionOverride,
		client:                    newEmbeddingHTTPClient(60 * time.Second),
	}, nil
}

type loreLatticeCloudEmbedRequest struct {
	Model                string   `json:"model"`
	Input                []string `json:"input"`
	Dimensions           int      `json:"dimensions,omitempty"`
	TruncatePromptTokens int      `json:"truncate_prompt_tokens,omitempty"`
}

type loreLatticeCloudEmbedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (e *LoreLatticeCloudEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	results, err := e.BatchEmbed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("lorelatticecloud embedder: empty response")
	}
	return results[0], nil
}

func (e *LoreLatticeCloudEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := loreLatticeCloudEmbedRequest{Model: e.effectiveModelName(), Input: texts}
	if e.supportsDimensionOverride && e.dimensions > 0 {
		reqBody.Dimensions = e.dimensions
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("lorelatticecloud embedder: marshal: %w", err)
	}

	requestID := uuid.New().String()
	headers := utils.Sign(e.appID, e.apiKey, requestID, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+loreLatticeCloudEmbedPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("lorelatticecloud embedder: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lorelatticecloud embedder: do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lorelatticecloud embedder: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lorelatticecloud embedder: status %d: %s", resp.StatusCode, string(respBytes))
	}

	var embedResp loreLatticeCloudEmbedResponse
	if err := json.Unmarshal(respBytes, &embedResp); err != nil {
		return nil, fmt.Errorf("lorelatticecloud embedder: unmarshal: %w", err)
	}

	result := make([][]float32, len(texts))
	for _, item := range embedResp.Data {
		if item.Index < len(result) {
			result[item.Index] = item.Embedding
		}
	}
	return result, nil
}

func (e *LoreLatticeCloudEmbedder) BatchEmbedWithPool(ctx context.Context, model Embedder, texts []string) ([][]float32, error) {
	return e.BatchEmbed(ctx, texts)
}

func (e *LoreLatticeCloudEmbedder) SetSupportsDimensionOverride(supported bool) {
	e.supportsDimensionOverride = supported
}

func (e *LoreLatticeCloudEmbedder) effectiveModelName() string {
	if e.remoteModelName != "" {
		return e.remoteModelName
	}
	return e.modelName
}

func (e *LoreLatticeCloudEmbedder) GetModelName() string { return e.modelName }
func (e *LoreLatticeCloudEmbedder) GetModelID() string   { return e.modelID }
func (e *LoreLatticeCloudEmbedder) GetDimensions() int   { return e.dimensions }
