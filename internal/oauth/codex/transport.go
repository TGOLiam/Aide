package codex

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// NewTransport creates a custom HTTP round-tripper that rewrites OpenAI API
// requests to the Codex API endpoint at chatgpt.com and injects the
// necessary authentication headers.
//
// The Codex API uses a single URL endpoint that accepts the OpenAI Responses
// API request format. This transport:
//  1. Strips the original Authorization header
//  2. Injects Authorization: Bearer {accessToken}
//  3. Injects ChatGPT-Account-Id header when accountID is set
//  4. Rewrites the URL path to the Codex endpoint
func NewTransport(accessToken, accountID string, next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &codexTransport{
		accessToken: accessToken,
		accountID:   accountID,
		next:        next,
	}
}

type codexTransport struct {
	accessToken string
	accountID   string
	next        http.RoundTripper
}

func (t *codexTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("codex: request is nil")
	}

	// Only intercept requests that look like OpenAI API calls.
	path := req.URL.Path
	if !strings.HasSuffix(path, "/v1/responses") &&
		!strings.HasSuffix(path, "/chat/completions") {
		return t.next.RoundTrip(req)
	}

	// Clone the request to avoid mutating the original.
	req = req.Clone(req.Context())

	// Strip the original Authorization header.
	req.Header.Del("Authorization")
	req.Header.Del("authorization")

	// Inject the Codex Bearer token.
	req.Header.Set("Authorization", "Bearer "+t.accessToken)

	// Inject the ChatGPT-Account-Id header when we have it.
	if t.accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", t.accountID)
	}

	// Set the User-Agent to match what ChatGPT expects.
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "aide/1.0")
	}

	// Rewrite the URL to the Codex API endpoint. The Codex API receives
	// the same payload as the OpenAI Responses API but at a fixed URL.
	req.URL.Scheme = "https"
	req.URL.Host = "chatgpt.com"
	req.URL.Path = "/backend-api/codex/responses"

	// Restore the body if present (it may have been consumed by earlier
	// inspection).
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("codex: failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		req.ContentLength = int64(len(bodyBytes))
	}

	return t.next.RoundTrip(req)
}
