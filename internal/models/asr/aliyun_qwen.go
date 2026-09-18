package asr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	secutils "github.com/Pototoooo/lorelattice/internal/utils"
)

// AliyunQwenASR implements Qwen3-ASR through Alibaba Model Studio's
// OpenAI-compatible chat/completions input_audio contract. Qwen3-ASR does not
// expose the /audio/transcriptions endpoint used by Whisper-compatible APIs.
type AliyunQwenASR struct {
	modelName string
	modelID   string
	apiKey    string
	baseURL   string
	language  string
	client    *http.Client
}

type aliyunQwenASRRequest struct {
	Model      string                 `json:"model"`
	Messages   []aliyunQwenASRMessage `json:"messages"`
	Stream     bool                   `json:"stream"`
	ASROptions *aliyunQwenASROptions  `json:"asr_options,omitempty"`
}

type aliyunQwenASRMessage struct {
	Role    string                   `json:"role"`
	Content []aliyunQwenASRInputPart `json:"content"`
}

type aliyunQwenASRInputPart struct {
	Type       string                  `json:"type"`
	InputAudio aliyunQwenASRAudioInput `json:"input_audio"`
}

type aliyunQwenASRAudioInput struct {
	Data string `json:"data"`
}

type aliyunQwenASROptions struct {
	Language string `json:"language,omitempty"`
}

type aliyunQwenASRResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewAliyunQwenASR(config *Config) (*AliyunQwenASR, error) {
	if config == nil {
		return nil, fmt.Errorf("ASR config is nil")
	}
	if err := validateASRBaseURL(config.BaseURL); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.ModelName) == "" {
		return nil, fmt.Errorf("model name is required")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("API key is required")
	}

	httpClient := newASRHTTPClient(asrDefaultTimeout)
	if len(config.CustomHeaders) > 0 {
		httpClient = secutils.WrapHTTPClientWithHeaders(httpClient, config.CustomHeaders)
	}

	return &AliyunQwenASR{
		modelName: config.ModelName,
		modelID:   config.ModelID,
		apiKey:    config.APIKey,
		baseURL:   strings.TrimRight(config.BaseURL, "/"),
		language:  config.Language,
		client:    httpClient,
	}, nil
}

func (s *AliyunQwenASR) Transcribe(
	ctx context.Context, audioBytes []byte, fileName string,
) (*TranscriptionResult, error) {
	if len(audioBytes) == 0 {
		return nil, fmt.Errorf("audio bytes are empty")
	}

	mimeType := audioMIMEType(audioBytes, fileName)
	dataURI := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(audioBytes)
	payload := aliyunQwenASRRequest{
		Model: s.modelName,
		Messages: []aliyunQwenASRMessage{{
			Role: "user",
			Content: []aliyunQwenASRInputPart{{
				Type:       "input_audio",
				InputAudio: aliyunQwenASRAudioInput{Data: dataURI},
			}},
		}},
		Stream: false,
	}
	if strings.TrimSpace(s.language) != "" {
		payload.ASROptions = &aliyunQwenASROptions{Language: s.language}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal Qwen ASR request: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create Qwen ASR request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Qwen ASR request failed: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read Qwen ASR response: %w", err)
	}

	var decoded aliyunQwenASRResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode Qwen ASR response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if decoded.Error != nil {
			return nil, fmt.Errorf("Qwen ASR API error: HTTP %d, code=%s, message=%s",
				resp.StatusCode, decoded.Error.Code, decoded.Error.Message)
		}
		return nil, fmt.Errorf("Qwen ASR API error: HTTP %d", resp.StatusCode)
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("Qwen ASR response contained no choices")
	}

	text := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if text == "" {
		return nil, fmt.Errorf("Qwen ASR response contained empty transcription")
	}
	return &TranscriptionResult{Text: text}, nil
}

func (s *AliyunQwenASR) GetModelName() string { return s.modelName }
func (s *AliyunQwenASR) GetModelID() string   { return s.modelID }

func audioMIMEType(audioBytes []byte, fileName string) string {
	if ext := strings.ToLower(filepath.Ext(fileName)); ext != "" {
		switch ext {
		case ".wav":
			return "audio/wav"
		case ".mp3":
			return "audio/mpeg"
		case ".m4a":
			return "audio/mp4"
		}
		if detected := mime.TypeByExtension(ext); detected != "" {
			return strings.Split(detected, ";")[0]
		}
	}
	if len(audioBytes) > 0 {
		if detected := http.DetectContentType(audioBytes); detected != "application/octet-stream" {
			return strings.Split(detected, ";")[0]
		}
	}
	return "audio/mpeg"
}

var _ ASR = (*AliyunQwenASR)(nil)
