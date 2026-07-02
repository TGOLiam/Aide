// Package codex provides OAuth authentication for the OpenAI Codex API.
// Codex is the backend API powering chatgpt.com, accessible to ChatGPT
// Plus, Pro, and Edu subscribers.
package codex

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/liamb/aide/internal/oauth"
)

// ProviderID is the identifier used for the Codex provider in config.
const ProviderID = "codex"

// APIEndpoint is the Codex API endpoint that proxies OpenAI Requests.
const APIEndpoint = "https://chatgpt.com/backend-api/codex"

// DefaultModels are the GPT models available through the Codex API.
// These are served at zero cost through a ChatGPT subscription.
var DefaultModels = []map[string]any{
	{
		"id":                    "gpt-5.5",
		"name":                  "GPT-5.5",
		"context_window":        400_000,
		"default_max_tokens":    128_000,
		"can_reason":            true,
		"reasoning_levels":      []string{"low", "medium", "high"},
		"supports_attachments":  true,
		"cost_per_1m_in":        0,
		"cost_per_1m_out":       0,
		"cost_per_1m_in_cached": 0,
		"cost_per_1m_out_cached": 0,
	},
	{
		"id":                    "gpt-5.4",
		"name":                  "GPT-5.4",
		"context_window":        200_000,
		"default_max_tokens":    128_000,
		"can_reason":            false,
		"supports_attachments":  true,
		"cost_per_1m_in":        0,
		"cost_per_1m_out":       0,
		"cost_per_1m_in_cached": 0,
		"cost_per_1m_out_cached": 0,
	},
	{
		"id":                    "gpt-5.4-mini",
		"name":                  "GPT-5.4 Mini",
		"context_window":        200_000,
		"default_max_tokens":    128_000,
		"can_reason":            false,
		"supports_attachments":  true,
		"cost_per_1m_in":        0,
		"cost_per_1m_out":       0,
		"cost_per_1m_in_cached": 0,
		"cost_per_1m_out_cached": 0,
	},
	{
		"id":                    "gpt-5.3-codex-spark",
		"name":                  "GPT-5.3 Codex Spark",
		"context_window":        200_000,
		"default_max_tokens":    128_000,
		"can_reason":            false,
		"supports_attachments":  true,
		"cost_per_1m_in":        0,
		"cost_per_1m_out":       0,
		"cost_per_1m_in_cached": 0,
		"cost_per_1m_out_cached": 0,
	},
}

// ProviderConfig returns a catwalk.Provider definition for the OpenAI Codex
// API. This is injected into the known providers list so it appears in the
// model selection UI without requiring manual configuration.
func ProviderConfig() catwalk.Provider {
	models := make([]catwalk.Model, len(DefaultModels))
	for i, m := range DefaultModels {
		data, _ := json.Marshal(m)
		_ = json.Unmarshal(data, &models[i])
	}
	return catwalk.Provider{
		Name:        "OpenAI Codex",
		ID:          catwalk.InferenceProvider(ProviderID),
		APIEndpoint: APIEndpoint,
		Type:        catwalk.TypeOpenAICompat,
		Models:      models,
	}
}

const (
	// OpenAI OAuth constants matching OpenCode's plugin.
	clientID      = "app_EMoamEEZ73f0CkXaXp7hrann"
	issuer        = "https://auth.openai.com"
	deviceAuthURL = "https://auth.openai.com/api/accounts/deviceauth/usercode"
	tokenURL      = "https://auth.openai.com/oauth/token"
	deviceTokenURL = "https://auth.openai.com/api/accounts/deviceauth/token"
)

// --- JWT parsing ---

// idTokenClaims represents the claims in the OpenAI id_token JWT.
type idTokenClaims struct {
	ChatGPTAccountID string `json:"chatgpt_account_id"`
	Organizations    []struct {
		ID string `json:"id"`
	} `json:"organizations"`
	Email string `json:"email"`
	Auth  *struct {
		ChatGPTAccountID string `json:"chatgpt_account_id"`
	} `json:"https://api.openai.com/auth"`
}

// extractAccountID extracts the ChatGPT account ID from a JWT token string.
func extractAccountID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims idTokenClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}
	if claims.ChatGPTAccountID != "" {
		return claims.ChatGPTAccountID
	}
	if claims.Auth != nil && claims.Auth.ChatGPTAccountID != "" {
		return claims.Auth.ChatGPTAccountID
	}
	if len(claims.Organizations) > 0 {
		return claims.Organizations[0].ID
	}
	return ""
}

// --- Token response types ---

// tokenResponse is the response from the OpenAI OAuth token endpoint.
type tokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// deviceAuthResponse is the response from the device auth initiation endpoint.
type deviceAuthResponse struct {
	DeviceAuthID string `json:"device_auth_id"`
	UserCode     string `json:"user_code"`
	Interval     string `json:"interval"`
	ExpiresIn    int    `json:"expires_in"`
	VerificationURL string `json:"verification_uri"`
}

// deviceTokenResponse is the response from polling the device auth token endpoint.
type deviceTokenResponse struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
}

// --- Device auth flow ---

// DeviceAuthResponse contains the result of initiating device auth.
type DeviceAuthResponse struct {
	DeviceAuthID    string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
}

// InitiateDeviceAuth starts the device authorization flow.
// Returns the device code, user code, verification URL, and expiry.
func InitiateDeviceAuth(ctx context.Context) (*deviceAuthResponse, error) {
	body := map[string]string{"client_id": clientID}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceAuthURL, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aide/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device auth failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result deviceAuthResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.VerificationURL == "" {
		result.VerificationURL = issuer + "/codex/device"
	}
	if result.ExpiresIn <= 0 {
		result.ExpiresIn = 300 // default 5 minutes if API doesn't provide it
	}
	return &result, nil
}

// PollForToken polls the device auth endpoint until the user completes authorization.
// Returns the OAuth tokens on success.
func PollForToken(ctx context.Context, deviceAuthID, userCode string, expiresIn int) (*oauth.Token, error) {
	if expiresIn <= 0 {
		expiresIn = 300 // default 5 minutes
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(expiresIn)*time.Second)
	defer cancel()

	pollInterval := 5 * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			token, err := pollOnce(ctx, deviceAuthID, userCode)
			if err != nil {
				return nil, err
			}
			if token != nil {
				return token, nil
			}
			// Keep polling
		}
	}
}

func pollOnce(ctx context.Context, deviceAuthID, userCode string) (*oauth.Token, error) {
	body := map[string]string{
		"device_auth_id": deviceAuthID,
		"user_code":      userCode,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceTokenURL, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aide/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		// Authorization pending, return nil to keep polling
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token poll failed: status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var dtr deviceTokenResponse
	if err := json.Unmarshal(respBody, &dtr); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Exchange the authorization code for tokens
	return exchangeAuthCode(ctx, dtr.AuthorizationCode, dtr.CodeVerifier)
}

func exchangeAuthCode(ctx context.Context, code, codeVerifier string) (*oauth.Token, error) {
	form := fmt.Sprintf(
		"grant_type=authorization_code&code=%s&redirect_uri=%s/deviceauth/callback&client_id=%s&code_verifier=%s",
		code, issuer, clientID, codeVerifier,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var tr tokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	t := &oauth.Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
		AccountID:    extractAccountID(tr.IDToken),
	}
	t.SetExpiresAt()

	return t, nil
}

// --- PKCE browser OAuth flow ---

const oauthPort = 1455
const oauthCallbackPath = "/auth/callback"
const oauthTimeout = 5 * time.Minute

// PKCESession holds state for a PKCE browser-based OAuth flow.
type PKCESession struct {
	Verifier    string
	Challenge   string
	State       string
	RedirectURI string
	server      *http.Server
	resultCh    chan *oauth.Token
	errCh       chan error
}

// GeneratePKCE creates a PKCE code verifier and its SHA-256 challenge.
func GeneratePKCE() (*PKCESession, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	verifierBytes := make([]byte, 43)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("generate verifier: %w", err)
	}
	verifier := make([]byte, 43)
	for i, b := range verifierBytes {
		verifier[i] = chars[int(b)%len(chars)]
	}

	hash := sha256.Sum256(verifier)
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	redirectURI := fmt.Sprintf("http://localhost:%d%s", oauthPort, oauthCallbackPath)

	return &PKCESession{
		Verifier:    string(verifier),
		Challenge:   challenge,
		State:       state,
		RedirectURI: redirectURI,
		resultCh:    make(chan *oauth.Token, 1),
		errCh:       make(chan error, 1),
	}, nil
}

// AuthorizeURL builds the OpenAI OAuth authorize URL for the PKCE flow.
func (s *PKCESession) AuthorizeURL() string {
	params := url.Values{
		"response_type":              {"code"},
		"client_id":                  {clientID},
		"redirect_uri":               {s.RedirectURI},
		"scope":                      {"openid profile email offline_access"},
		"code_challenge":             {s.Challenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {s.State},
		"originator":                 {"aide"},
	}
	return issuer + "/oauth/authorize?" + params.Encode()
}

// StartLocalServer starts an HTTP server on localhost to catch the OAuth callback.
// Blocks until the server is ready or fails to start.
func (s *PKCESession) StartLocalServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc(oauthCallbackPath, s.handleCallback)
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		s.errCh <- fmt.Errorf("login cancelled")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Login cancelled"))
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", oauthPort),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("start local server: %w", err)
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

// StopLocalServer shuts down the local HTTP server.
func (s *PKCESession) StopLocalServer() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
		s.server = nil
	}
}

// WaitForResult blocks until the OAuth callback is received or the timeout expires.
func (s *PKCESession) WaitForResult(timeout time.Duration) (*oauth.Token, error) {
	if timeout <= 0 {
		timeout = oauthTimeout
	}
	select {
	case token := <-s.resultCh:
		return token, nil
	case err := <-s.errCh:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("authorization timed out after %v", timeout)
	}
}

func (s *PKCESession) exchangeToken(ctx context.Context, code string, w http.ResponseWriter) {
	form := fmt.Sprintf(
		"grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&code_verifier=%s",
		code, s.RedirectURI, clientID, s.Verifier,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form))
	if err != nil {
		s.errCh <- fmt.Errorf("token exchange: %w", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.errCh <- fmt.Errorf("token exchange: %w", err)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		s.errCh <- fmt.Errorf("token exchange: %w", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		s.errCh <- fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	var tr tokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		s.errCh <- fmt.Errorf("token exchange: %w", err)
		return
	}

	t := &oauth.Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
		AccountID:    extractAccountID(tr.IDToken),
	}
	t.SetExpiresAt()
	s.resultCh <- t
}

func (s *PKCESession) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	if errorParam != "" {
		msg := errorDesc
		if msg == "" {
			msg = errorParam
		}
		s.errCh <- fmt.Errorf("authorization failed: %s", msg)
		writeOAuthResponse(w, oauthErrorPage(msg))
		return
	}

	if state != s.State {
		s.errCh <- fmt.Errorf("state mismatch — possible CSRF attack")
		writeOAuthResponse(w, oauthErrorPage("Invalid state"))
		return
	}

	if code == "" {
		s.errCh <- fmt.Errorf("no authorization code received")
		writeOAuthResponse(w, oauthErrorPage("Missing authorization code"))
		return
	}

	go s.exchangeToken(r.Context(), code, w)

	writeOAuthResponse(w, oauthSuccessPage())
}

func writeOAuthResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

func oauthSuccessPage() string {
	return `<!DOCTYPE html><html><body><h2>✅ Authorization successful</h2><p>You can close this tab and return to Aide.</p></body></html>`
}

func oauthErrorPage(msg string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><body><h2>❌ Authorization failed</h2><p>%s</p></body></html>`, msg)
}

// --- Token refresh ---

// RefreshToken exchanges a refresh token for a new set of tokens.
func RefreshToken(ctx context.Context, refreshToken string) (*oauth.Token, error) {
	form := fmt.Sprintf(
		"grant_type=refresh_token&refresh_token=%s&client_id=%s",
		refreshToken, clientID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var tr tokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	t := &oauth.Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
		AccountID:    extractAccountID(tr.IDToken),
	}
	t.SetExpiresAt()

	return t, nil
}
