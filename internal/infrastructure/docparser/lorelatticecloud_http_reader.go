package docparser

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Pototoooo/lorelattice/internal/logger"

	"github.com/Pototoooo/lorelattice/internal/models/utils"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/google/uuid"
)

var (
	loreLatticeCloudReaderBaseURL = strings.TrimRight(
		strings.TrimSpace(os.Getenv("LORELATTICE_CLOUD_BASE_URL")),
		"/",
	) + "/api/v1/doc"
)

// LoreLatticeCloudSignedDocumentReader implements the docreader HTTP protocol with LoreLatticeCloud signing.
type LoreLatticeCloudSignedDocumentReader struct {
	appID               string
	apiKey              string
	client              *http.Client
	initialPollInterval time.Duration
	maxPollInterval     time.Duration
	pollTimeout         time.Duration
}

func NewLoreLatticeCloudSignedDocumentReader(appID, apiKey string) (*LoreLatticeCloudSignedDocumentReader, error) {
	if strings.TrimSpace(os.Getenv("LORELATTICE_CLOUD_BASE_URL")) == "" {
		return nil, fmt.Errorf("LORELATTICE_CLOUD_BASE_URL is not configured")
	}
	if appID == "" {
		return nil, fmt.Errorf("LoreLatticeCloud appID is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("LoreLatticeCloud apiKey is required")
	}
	return &LoreLatticeCloudSignedDocumentReader{
		appID:               appID,
		apiKey:              apiKey,
		initialPollInterval: 500 * time.Millisecond,
		maxPollInterval:     10 * time.Second,
		pollTimeout:         20 * time.Minute,
		client: &http.Client{
			Timeout: 500 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
	}, nil
}

func (p *LoreLatticeCloudSignedDocumentReader) Reconnect(addr string) error {
	return nil
}

func (p *LoreLatticeCloudSignedDocumentReader) IsConnected() bool { return true }

func (p *LoreLatticeCloudSignedDocumentReader) ListEngines(ctx context.Context, overrides map[string]string) ([]types.ParserEngineInfo, error) {
	return []types.ParserEngineInfo{{
		Name:        LoreLatticeCloudEngineName,
		Description: "LoreLatticeCloud signed docreader",
		FileTypes:   []string{"docx", "doc", "pdf", "md", "markdown", "xlsx", "xls", "pptx", "ppt"},
		Available:   true,
	}}, nil
}

func (p *LoreLatticeCloudSignedDocumentReader) Read(ctx context.Context, req *types.ReadRequest) (*types.ReadResult, error) {
	logger.Infof(ctx, "[LoreLatticeCloud] read start file=%q type=%q engine=%q hasURL=%v contentLen=%d requestID=%q",
		req.FileName, req.FileType, req.ParserEngine, strings.TrimSpace(req.URL) != "", len(req.FileContent), req.RequestID)
	body := httpReadRequest{
		FileName:  req.FileName,
		FileType:  req.FileType,
		URL:       req.URL,
		Title:     req.Title,
		RequestID: req.RequestID,
		Config: &httpReadConfig{
			ParserEngine:          req.ParserEngine,
			ParserEngineOverrides: req.ParserEngineOverrides,
		},
	}
	if len(req.FileContent) > 0 {
		body.FileContent = base64.StdEncoding.EncodeToString(req.FileContent)
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		logger.Errorf(context.Background(), "[LoreLatticeCloud] marshal read request: %v", err)
		return nil, fmt.Errorf("http marshal read request: %w", err)
	}
	httpReq, err := p.newSignedRequest(ctx, http.MethodPost, loreLatticeCloudReaderBaseURL+"/reader", jsonBody)
	if err != nil {
		logger.Errorf(context.Background(), "[LoreLatticeCloud] signed read request: %v", err)
		return nil, err
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		logger.Errorf(context.Background(), "[LoreLatticeCloud] http read request failed: %v", err)
		return nil, fmt.Errorf("http read failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.Errorf(context.Background(), "[LoreLatticeCloud] http read unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("http read status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var submit loreLatticeCloudAsyncSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submit); err != nil {
		logger.Errorf(context.Background(), "[LoreLatticeCloud] decode read submit response: %v", err)
		return nil, fmt.Errorf("http decode read submit response: %w", err)
	}
	if strings.TrimSpace(submit.TaskID) == "" {
		logger.Errorf(context.Background(), "[LoreLatticeCloud] submit response missing task_id (status=%q message=%q)", submit.Status, submit.Message)
		return nil, fmt.Errorf("lorelatticecloud docreader submit response missing task_id")
	}
	logger.Infof(ctx, "[LoreLatticeCloud] task submitted task_id=%s file=%q type=%q", submit.TaskID, req.FileName, req.FileType)
	return p.pollTaskResult(ctx, submit.TaskID)
}

type loreLatticeCloudAsyncSubmitResponse struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
}

type loreLatticeCloudAsyncTaskResponse struct {
	TaskID    string            `json:"task_id"`
	Status    string            `json:"status"`
	Message   string            `json:"message"`
	Progress  float64           `json:"progress"`
	Result    *httpReadResponse `json:"result"`
	Error     string            `json:"error"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

func (p *LoreLatticeCloudSignedDocumentReader) pollTaskResult(ctx context.Context, taskID string) (*types.ReadResult, error) {
	pollCtx := ctx
	if _, ok := ctx.Deadline(); !ok && p.pollTimeout > 0 {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, p.pollTimeout)
		defer cancel()
	}
	statusURL := loreLatticeCloudReaderBaseURL + "/" + taskID
	currentInterval := p.initialPollInterval
	for {
		httpReq, err := p.newSignedRequest(pollCtx, http.MethodGet, statusURL, nil)
		if err != nil {
			logger.Errorf(context.Background(), "[LoreLatticeCloud] poll signed request task_id=%s: %v", taskID, err)
			return nil, err
		}
		resp, err := p.client.Do(httpReq)
		if err != nil {
			logger.Errorf(context.Background(), "[LoreLatticeCloud] http poll task_id=%s failed: %v", taskID, err)
			return nil, fmt.Errorf("http poll task failed: %w", err)
		}
		var taskResp loreLatticeCloudAsyncTaskResponse
		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				err = fmt.Errorf("http poll task status %d: %s", resp.StatusCode, string(bodyBytes))
				logger.Errorf(context.Background(), "[LoreLatticeCloud] poll task_id=%s status %d: %s", taskID, resp.StatusCode, string(bodyBytes))
				return
			}
			if decodeErr := json.NewDecoder(resp.Body).Decode(&taskResp); decodeErr != nil {
				err = fmt.Errorf("http decode task response: %w", decodeErr)
				logger.Errorf(context.Background(), "[LoreLatticeCloud] poll task_id=%s decode response: %v", taskID, decodeErr)
			}
		}()
		if err != nil {
			return nil, err
		}
		switch taskResp.Status {
		case "completed":
			if taskResp.Result == nil {
				logger.Infof(ctx, "[LoreLatticeCloud] task_id=%s completed with no result payload", taskID)
				return &types.ReadResult{}, nil
			}
			res := fromHTTPReadResponse(taskResp.Result)
			if res.Error != "" {
				logger.Errorf(ctx, "[LoreLatticeCloud] task_id=%s completed with result.error: %s", taskID, res.Error)
			} else {
				logger.Debugf(ctx, "[LoreLatticeCloud] task_id=%s completed ok markdownLen=%d", taskID, len(res.MarkdownContent))
			}
			return res, nil
		case "failed":
			if taskResp.Error != "" {
				logger.Errorf(context.Background(), "[LoreLatticeCloud] task_id=%s failed: %s", taskID, taskResp.Error)
				return nil, fmt.Errorf("lorelatticecloud docreader task failed: %s", taskResp.Error)
			}
			logger.Errorf(context.Background(), "[LoreLatticeCloud] task_id=%s failed: %s", taskID, taskResp.Message)
			return nil, fmt.Errorf("lorelatticecloud docreader task failed: %s", taskResp.Message)
		case "cancelled":
			if taskResp.Error != "" {
				logger.Errorf(context.Background(), "[LoreLatticeCloud] task_id=%s cancelled: %s", taskID, taskResp.Error)
				return nil, fmt.Errorf("lorelatticecloud docreader task cancelled: %s", taskResp.Error)
			}
			logger.Errorf(context.Background(), "[LoreLatticeCloud] task_id=%s cancelled", taskID)
			return nil, fmt.Errorf("lorelatticecloud docreader task cancelled")
		}
		if err := pollCtx.Err(); err != nil {
			logger.Errorf(ctx, "[LoreLatticeCloud] poll task_id=%s aborted before sleep: %v", taskID, err)
			return nil, err
		}

		// Exponential backoff: multiply by 1.5 each time, cap at maxPollInterval
		select {
		case <-pollCtx.Done():
			logger.Errorf(ctx, "[LoreLatticeCloud] poll task_id=%s stopped: %v", taskID, pollCtx.Err())
			return nil, pollCtx.Err()
		case <-time.After(currentInterval):
			// Update interval for next iteration
			nextInterval := time.Duration(float64(currentInterval) * 1.5)
			if nextInterval > p.maxPollInterval {
				nextInterval = p.maxPollInterval
			}
			currentInterval = nextInterval
		}
	}
}

func (p *LoreLatticeCloudSignedDocumentReader) newSignedRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	requestID := uuid.New().String()
	if len(body) == 0 {
		body = []byte("{}")
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		logger.Errorf(context.Background(), "[LoreLatticeCloud] http new request %s %s: %v", method, url, err)
		return nil, fmt.Errorf("http new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.ContentLength = int64(len(body))
	for k, v := range utils.Sign(p.appID, p.apiKey, requestID, string(body)) {
		httpReq.Header.Set(k, v)
	}
	return httpReq, nil
}
