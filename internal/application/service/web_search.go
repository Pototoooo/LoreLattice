package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// WebSearchProvider defines the interface for web search providers
type WebSearchProvider interface {
	Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error)
	Name() string
}

// WebSearchService provides web search functionality
type WebSearchService struct {
	providers map[string]WebSearchProvider
	config    *config.WebSearchConfig
}

// Search performs web search using the specified provider
// This method implements the interface expected by PluginSearch
func (s *WebSearchService) Search(ctx context.Context, config *types.WebSearchConfig, query string) ([]*types.WebSearchResult, error) {
	if config == nil {
		return nil, fmt.Errorf("web search config is required")
	}

	provider, ok := s.providers[config.Provider]
	if !ok {
		return nil, fmt.Errorf("web search provider %s is not available", config.Provider)
	}

	// Set API key for providers that need it
	if config.APIKey != "" {
		s.setProviderAPIKey(config.Provider, provider, config)
	}

	// Set timeout
	timeout := time.Duration(s.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Perform search
	results, err := provider.Search(ctx, query, config.MaxResults, config.IncludeDate)
	if err != nil {
		return nil, fmt.Errorf("web search failed: %w", err)
	}

	// Apply blacklist filtering
	results = s.filterBlacklist(results, config.Blacklist)

	// Apply compression if needed
	if config.CompressionMethod != "none" && config.CompressionMethod != "" {
		// Compression will be handled later in the integration layer
		// For now, we just return the results
	}

	return results, nil
}

// NewWebSearchService creates a new web search service
func NewWebSearchService(cfg *config.Config) (*WebSearchService, error) {
	if cfg.WebSearch == nil {
		return nil, fmt.Errorf("web search config is not available")
	}

	service := &WebSearchService{
		providers: make(map[string]WebSearchProvider),
		config:    cfg.WebSearch,
	}

	// Initialize providers based on config
	for _, providerConfig := range cfg.WebSearch.Providers {
		var provider WebSearchProvider
		var err error

		switch providerConfig.ID {
		case "kuaisou":
			provider, err = NewKuaisouProvider(providerConfig)
		case "baidu":
			provider, err = NewBaiduProvider(providerConfig)
		case "google":
			provider, err = NewGoogleProvider(providerConfig)
		case "bing":
			provider, err = NewBingProvider(providerConfig)
		case "bocha":
			provider, err = NewBochaProvider(providerConfig)
		case "zhipu":
			provider, err = NewZhipuProvider(providerConfig)
		case "tavily":
			provider, err = NewTavilyProvider(providerConfig)
		case "searxng":
			provider, err = NewSearxngProvider(providerConfig)
		case "exa":
			provider, err = NewExaProvider(providerConfig)
		default:
			logger.Warnf(context.Background(), "Unknown web search provider: %s", providerConfig.ID)
			continue
		}

		if err != nil {
			logger.Warnf(context.Background(), "Failed to initialize provider %s: %v", providerConfig.ID, err)
			continue
		}

		service.providers[providerConfig.ID] = provider
		logger.Infof(context.Background(), "Initialized web search provider: %s", providerConfig.ID)
	}

	return service, nil
}

// setProviderAPIKey sets the API key for a provider based on its type
func (s *WebSearchService) setProviderAPIKey(providerID string, provider WebSearchProvider, config *types.WebSearchConfig) {
	switch p := provider.(type) {
	case *KuaisouProvider:
		p.SetAPIKey(config.APIKey)
	case *BochaProvider:
		p.SetAPIKey(config.APIKey)
	case *ZhipuProvider:
		p.SetAPIKey(config.APIKey)
	case *TavilyProvider:
		p.SetAPIKey(config.APIKey)
	case *ExaProvider:
		p.SetAPIKey(config.APIKey)
	case *GoogleProvider:
		// Google needs both API key and search engine ID
		// For now, we'll use API key as search engine ID if not provided separately
		// This can be extended later to support separate fields
		p.SetAPIKey(config.APIKey, config.APIKey) // TODO: Add search engine ID to config
	}
}

// filterBlacklist filters results based on blacklist rules
func (s *WebSearchService) filterBlacklist(results []*types.WebSearchResult, blacklist []string) []*types.WebSearchResult {
	if len(blacklist) == 0 {
		return results
	}

	filtered := make([]*types.WebSearchResult, 0, len(results))

	for _, result := range results {
		shouldFilter := false

		for _, rule := range blacklist {
			if s.matchesBlacklistRule(result.URL, rule) {
				shouldFilter = true
				break
			}
		}

		if !shouldFilter {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// matchesBlacklistRule checks if a URL matches a blacklist rule
// Supports both pattern matching (e.g., *://*.example.com/*) and regex patterns (e.g., /example\.(net|org)/)
func (s *WebSearchService) matchesBlacklistRule(url, rule string) bool {
	// Check if it's a regex pattern (starts and ends with /)
	if strings.HasPrefix(rule, "/") && strings.HasSuffix(rule, "/") {
		pattern := rule[1 : len(rule)-1]
		matched, err := regexp.MatchString(pattern, url)
		if err != nil {
			logger.Warnf(context.Background(), "Invalid regex pattern in blacklist: %s, error: %v", rule, err)
			return false
		}
		return matched
	}

	// Pattern matching (e.g., *://*.example.com/*)
	pattern := strings.ReplaceAll(rule, "*", ".*")
	pattern = "^" + pattern + "$"
	matched, err := regexp.MatchString(pattern, url)
	if err != nil {
		logger.Warnf(context.Background(), "Invalid pattern in blacklist: %s, error: %v", rule, err)
		return false
	}
	return matched
}

// ConvertWebSearchResults converts WebSearchResult to SearchResult
func ConvertWebSearchResults(webResults []*types.WebSearchResult) []*types.SearchResult {
	results := make([]*types.SearchResult, 0, len(webResults))

	for i, webResult := range webResults {
		// Use URL as ChunkID for web search results
		chunkID := webResult.URL
		if chunkID == "" {
			chunkID = fmt.Sprintf("web_search_%d", i)
		}

		// Combine title and snippet as content
		content := webResult.Title
		if webResult.Snippet != "" {
			if content != "" {
				content += "\n\n" + webResult.Snippet
			} else {
				content = webResult.Snippet
			}
		}
		if webResult.Content != "" {
			if content != "" {
				content += "\n\n" + webResult.Content
			} else {
				content = webResult.Content
			}
		}

		// Set a default score for web search results (0.6, indicating medium relevance)
		score := 0.6

		result := &types.SearchResult{
			ID:             chunkID,
			Content:        content,
			KnowledgeID:    "", // Web search results don't have knowledge ID
			ChunkIndex:     0,
			KnowledgeTitle: webResult.Title,
			StartAt:        0,
			EndAt:          len(content),
			Seq:            i,
			Score:          score,
			MatchType:      types.MatchTypeWebSearch,
			SubChunkID:     []string{},
			Metadata: map[string]string{
				"url":     webResult.URL,
				"source":  webResult.Source,
				"title":   webResult.Title,
				"snippet": webResult.Snippet,
			},
			ChunkType:         "web_search",
			ParentChunkID:     "",
			ImageInfo:         "",
			KnowledgeFilename: "",
			KnowledgeSource:   "web_search",
		}

		// Add published date to metadata if available
		if webResult.PublishedAt != nil {
			result.Metadata["published_at"] = webResult.PublishedAt.Format(time.RFC3339)
		}

		results = append(results, result)
	}

	return results
}
