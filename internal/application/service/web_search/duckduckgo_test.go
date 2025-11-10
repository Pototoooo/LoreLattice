package web_search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
)

// testRoundTripper rewrites outgoing requests that target DuckDuckGo hosts
// to the provided test server, preserving path and query.
type testRoundTripper struct {
	base *url.URL
	next http.RoundTripper
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only rewrite requests to duckduckgo hosts used by the provider
	if req.URL.Host == "html.duckduckgo.com" || req.URL.Host == "api.duckduckgo.com" {
		cloned := *req
		u := *req.URL
		u.Scheme = t.base.Scheme
		u.Host = t.base.Host
		// Keep original path; our test server handlers should register for the same paths.
		cloned.URL = &u
		req = &cloned
	}
	return t.next.RoundTrip(req)
}

func newTestClient(ts *httptest.Server) *http.Client {
	baseURL, _ := url.Parse(ts.URL)
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &testRoundTripper{
			base: baseURL,
			next: http.DefaultTransport,
		},
	}
}

func TestDuckDuckGoProvider_Name(t *testing.T) {
	p, _ := NewDuckDuckGoProvider(config.WebSearchProviderConfig{})
	if p.Name() != "duckduckgo" {
		t.Fatalf("expected provider name duckduckgo, got %s", p.Name())
	}
}

func TestDuckDuckGoProvider_Search_HTMLSuccess(t *testing.T) {
	// Minimal HTML page with two results, matching selectors used in searchHTML
	html := `
<html>
  <body>
    <div class="web-result">
      <a class="result__a" href="https://duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage1&rut=">Example One</a>
      <div class="result__snippet">Snippet one</div>
    </div>
    <div class="web-result">
      <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.org%2Fpage2&rut=">Example Two</a>
      <div class="result__snippet">Snippet two</div>
    </div>
  </body>
</html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Provider requests GET https://html.duckduckgo.com/html/?q=...&kl=...
		if r.URL.Path == "/html/" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		t.Fatalf("unexpected request path: %s", r.URL.Path)
	}))
	defer ts.Close()

	// Build provider and inject our test client
	prov, _ := NewDuckDuckGoProvider(config.WebSearchProviderConfig{})
	dp := prov.(*DuckDuckGoProvider)
	dp.client = newTestClient(ts)

	ctx := context.Background()
	results, err := dp.Search(ctx, "weknora", 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Example One" || !strings.HasPrefix(results[0].URL, "https://example.com/") || results[0].Snippet != "Snippet one" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].Title != "Example Two" || !strings.HasPrefix(results[1].URL, "https://example.org/") || results[1].Snippet != "Snippet two" {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
}

func TestDuckDuckGoProvider_Search_APIFallback(t *testing.T) {
	// Simulate HTML returning non-OK to force API fallback, then a minimal API JSON
	apiResp := struct {
		AbstractText string `json:"AbstractText"`
		AbstractURL  string `json:"AbstractURL"`
		Heading      string `json:"Heading"`
		Results      []struct {
			FirstURL string `json:"FirstURL"`
			Text     string `json:"Text"`
		} `json:"Results"`
	}{
		AbstractText: "Abstract snippet",
		AbstractURL:  "https://example.com/abstract",
		Heading:      "Abstract Heading",
		Results: []struct {
			FirstURL string `json:"FirstURL"`
			Text     string `json:"Text"`
		}{
			{FirstURL: "https://example.net/x", Text: "Title X - Detail X"},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/html/":
			// Force fallback by returning 500
			w.WriteHeader(http.StatusInternalServerError)
		default:
			// API endpoint path "/"
			w.Header().Set("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			_ = enc.Encode(apiResp)
		}
	}))
	defer ts.Close()

	prov, _ := NewDuckDuckGoProvider(config.WebSearchProviderConfig{})
	dp := prov.(*DuckDuckGoProvider)
	dp.client = newTestClient(ts)

	ctx := context.Background()
	results, err := dp.Search(ctx, "weknora", 3, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected some results from API fallback")
	}
	if results[0].URL != "https://example.com/abstract" || results[0].Title != "Abstract Heading" {
		t.Fatalf("unexpected first API result: %+v", results[0])
	}
}
