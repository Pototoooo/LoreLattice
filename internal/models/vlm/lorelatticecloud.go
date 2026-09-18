package vlm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Pototoooo/lorelattice/internal/logger"
	"github.com/Pototoooo/lorelattice/internal/models/utils"
	"github.com/google/uuid"
)

const loreLatticeCloudVLMPath = "/api/v1/chat/completions"

// LoreLatticeCloudVLM implements VLM via the LoreLatticeCloud API.
type LoreLatticeCloudVLM struct {
	modelName       string
	remoteModelName string
	modelID         string
	appID           string
	apiKey          string
	baseURL         string
	client          *http.Client
}

// NewLoreLatticeCloudVLM creates a LoreLatticeCloud-backed VLM instance.
func NewLoreLatticeCloudVLM(config *Config) (*LoreLatticeCloudVLM, error) {
	if config.AppID == "" {
		return nil, fmt.Errorf("LoreLatticeCloud VLM: AppID is required")
	}
	if config.AppSecret == "" {
		return nil, fmt.Errorf("LoreLatticeCloud VLM: AppSecret is required")
	}
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if err := validateVLMBaseURL(baseURL); err != nil {
		return nil, err
	}
	remoteModelName := ""
	if config.Extra != nil {
		if v, ok := config.Extra["remote_model_name"]; ok {
			if vs, ok := v.(string); ok {
				remoteModelName = strings.TrimSpace(vs)
			}
		}
	}
	return &LoreLatticeCloudVLM{
		modelName:       config.ModelName,
		remoteModelName: remoteModelName,
		modelID:         config.ModelID,
		appID:           config.AppID,
		apiKey:          config.AppSecret,
		baseURL:         baseURL,
		client:          newVLMHTTPClient(vlmHTTPTimeout()),
	}, nil
}

type loreLatticeCloudVLMContentPart struct {
	Type     string                       `json:"type"`
	Text     string                       `json:"text,omitempty"`
	ImageURL *loreLatticeCloudVLMImageURL `json:"image_url,omitempty"`
}

type loreLatticeCloudVLMImageURL struct {
	URL string `json:"url"`
}

type loreLatticeCloudVLMMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type loreLatticeCloudVLMRequest struct {
	Model       string                       `json:"model"`
	Messages    []loreLatticeCloudVLMMessage `json:"messages"`
	MaxTokens   int                          `json:"max_tokens,omitempty"`
	Temperature float64                      `json:"temperature,omitempty"`
	Stream      bool                         `json:"stream"`
}

type loreLatticeCloudVLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Predict sends images with a text prompt to the LoreLatticeCloud API.
func (v *LoreLatticeCloudVLM) Predict(ctx context.Context, imgBytesList [][]byte, prompt string) (string, error) {
	var parts []loreLatticeCloudVLMContentPart

	parts = append(parts, loreLatticeCloudVLMContentPart{
		Type: "text",
		Text: prompt,
	})

	for _, imgBytes := range imgBytesList {
		if len(imgBytes) > 0 {
			mimeType := detectImageMIME(imgBytes)
			b64 := base64.StdEncoding.EncodeToString(imgBytes)
			dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
			parts = append(parts, loreLatticeCloudVLMContentPart{
				Type: "image_url",
				ImageURL: &loreLatticeCloudVLMImageURL{
					URL: dataURI,
				},
			})
		}
	}

	reqBody := loreLatticeCloudVLMRequest{
		Model: v.effectiveModelName(),
		Messages: []loreLatticeCloudVLMMessage{
			{
				Role:    "user",
				Content: parts,
			},
		},
		MaxTokens:   defaultMaxToks,
		Temperature: float64(defaultTemp),
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("lorelatticecloud VLM: marshal: %w", err)
	}

	requestID := uuid.New().String()
	headers := utils.Sign(v.appID, v.apiKey, requestID, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+loreLatticeCloudVLMPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("lorelatticecloud VLM: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, hv := range headers {
		req.Header.Set(k, hv)
	}

	totalImageSize := 0
	for _, img := range imgBytesList {
		totalImageSize += len(img)
	}
	logger.Infof(ctx, "[VLM] Calling LoreLatticeCloud API, model=%s, baseURL=%s, numImages=%d, totalImageSize=%d",
		v.effectiveModelName(), v.baseURL, len(imgBytesList), totalImageSize)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lorelatticecloud VLM: do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("lorelatticecloud VLM: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lorelatticecloud VLM: status %d: %s", resp.StatusCode, string(respBytes))
	}

	var vlmResp loreLatticeCloudVLMResponse
	if err := json.Unmarshal(respBytes, &vlmResp); err != nil {
		return "", fmt.Errorf("lorelatticecloud VLM: unmarshal: %w", err)
	}
	if len(vlmResp.Choices) == 0 {
		return "", fmt.Errorf("lorelatticecloud VLM: no choices in response")
	}

	content := vlmResp.Choices[0].Message.Content
	logger.Infof(ctx, "[VLM] LoreLatticeCloud response received, len=%d", len(content))
	return content, nil
}

func (v *LoreLatticeCloudVLM) effectiveModelName() string {
	if v.remoteModelName != "" {
		return v.remoteModelName
	}
	return v.modelName
}

func (v *LoreLatticeCloudVLM) GetModelName() string { return v.modelName }
func (v *LoreLatticeCloudVLM) GetModelID() string   { return v.modelID }
