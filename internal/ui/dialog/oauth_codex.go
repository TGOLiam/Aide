package dialog

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"github.com/liamb/aide/internal/config"
	"github.com/liamb/aide/internal/oauth/codex"
	"github.com/liamb/aide/internal/ui/common"
	"github.com/pkg/browser"
)

// NewOAuthCodex creates a new OAuth dialog for OpenAI Codex authentication.
// It uses the PKCE browser-based OAuth flow (opens a browser window for the
// user to log in directly, rather than showing a device code).
func NewOAuthCodex(
	com *common.Common,
	isOnboarding bool,
	provider catwalk.Provider,
	model config.SelectedModel,
	modelType config.SelectedModelType,
) (*OAuth, tea.Cmd) {
	return newOAuth(com, isOnboarding, provider, model, modelType, &OAuthCodex{})
}

// OAuthCodex implements the OAuthProvider interface for OpenAI Codex.
// It uses the PKCE browser-based OAuth flow: opens the user's browser to
// auth.openai.com, starts a local web server to catch the callback, and
// exchanges the auth code for tokens.
type OAuthCodex struct {
	session   *codex.PKCESession
	cancelFunc func()
}

var _ OAuthProvider = (*OAuthCodex)(nil)

func (m *OAuthCodex) name() string {
	return "OpenAI Codex"
}

func (m *OAuthCodex) initiateAuth() tea.Msg {
	// Generate PKCE challenge and start local server.
	session, err := codex.GeneratePKCE()
	if err != nil {
		return ActionOAuthErrored{Error: fmt.Errorf("failed to initiate Codex auth: %w", err)}
	}
	m.session = session

	if err := session.StartLocalServer(); err != nil {
		return ActionOAuthErrored{Error: fmt.Errorf("failed to start local auth server: %w", err)}
	}

	authURL := session.AuthorizeURL()

	// Open the browser to the authorization URL.
	if err := browser.OpenURL(authURL); err != nil {
		// Browser didn't open, but we can still show the URL to the user.
		return ActionInitiateOAuth{
			DeviceCode:      "",
			UserCode:        authURL,
			VerificationURL: authURL,
			ExpiresIn:       300,
			Interval:        1,
		}
	}

	return ActionInitiateOAuth{
		DeviceCode:      "",
		UserCode:        "Browser opened",
		VerificationURL: authURL,
		ExpiresIn:       300,
		Interval:        1,
	}
}

func (m *OAuthCodex) startPolling(deviceCode string, expiresIn int) tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return ActionOAuthErrored{Error: fmt.Errorf("Codex auth session not initialized")}
		}

		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		token, err := m.session.WaitForResult(5 * time.Minute)
		if err != nil {
			if ctx.Err() != nil {
				return nil // cancelled, don't report error
			}
			return ActionOAuthErrored{Error: fmt.Errorf("Codex authorization failed: %w", err)}
		}

		return ActionCompleteOAuth{Token: token}
	}
}

func (m *OAuthCodex) stopPolling() tea.Msg {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	if m.session != nil {
		m.session.StopLocalServer()
	}
	return nil
}
