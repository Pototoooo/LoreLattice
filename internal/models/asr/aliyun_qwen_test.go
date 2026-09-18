package asr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewASRRoutesQwen3ASRToAliyunChatContract(t *testing.T) {
	got, err := newASR(&Config{
		Source:    "remote",
		Provider:  "aliyun",
		BaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
		ModelName: "qwen3-asr-flash",
		APIKey:    "test-key",
	})
	if err != nil {
		t.Fatalf("newASR: %v", err)
	}
	if _, ok := got.(*AliyunQwenASR); !ok {
		t.Fatalf("expected *AliyunQwenASR, got %T", got)
	}
}

func TestAliyunQwenASRTranscribeUsesInputAudio(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/compatible-mode/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing bearer auth")
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != "qwen3-asr-flash" {
			t.Fatalf("unexpected model: %v", payload["model"])
		}
		if !strings.Contains(string(body), `"type":"input_audio"`) ||
			!strings.Contains(string(body), "data:audio/wav;base64,") {
			t.Fatalf("missing input_audio data URI: %s", string(body))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"测试成功"}}]}`)),
		}, nil
	})}

	asr := &AliyunQwenASR{
		modelName: "qwen3-asr-flash",
		modelID:   "asr-1",
		apiKey:    "test-key",
		baseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
		language:  "zh",
		client:    client,
	}
	result, err := asr.Transcribe(context.Background(), []byte("RIFFfake-wave"), "sample.wav")
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if result.Text != "测试成功" {
		t.Fatalf("unexpected text: %q", result.Text)
	}
}
