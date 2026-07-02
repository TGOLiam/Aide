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
)

// NewOAuthCodex creates a new OAuth dialog for OpenAI Codex authentication.
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
// It uses the OpenAI device auth flow to obtain tokens for the Codex API.
type OAuthCodex struct {
	deviceAuthID string
	userCode     string
	cancelFunc   func()
}

var _ OAuthProvider = (*OAuthCodex)(nil)

func (m *OAuthCodex) name() string {
	return "OpenAI Codex"
}

func (m *OAuthCodex) initiateAuth() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := codex.InitiateDeviceAuth(ctx)
	if err != nil {
		return ActionOAuthErrored{Error: fmt.Errorf("failed to initiate Codex device auth: %w", err)}
	}

	m.deviceAuthID = resp.DeviceAuthID
	m.userCode = resp.UserCode

	return ActionInitiateOAuth{
		DeviceCode:      resp.DeviceAuthID,
		UserCode:        resp.UserCode,
		VerificationURL: resp.VerificationURL,
		ExpiresIn:       resp.ExpiresIn,
		Interval:        5,
	}
}

func (m *OAuthCodex) startPolling(deviceCode string, expiresIn int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		token, err := codex.PollForToken(ctx, m.deviceAuthID, m.userCode, expiresIn)
		if err != nil {
			if ctx.Err() != nil {
				return nil // cancelled, don't report error
			}
			return ActionOAuthErrored{Error: err}
		}

		return ActionCompleteOAuth{Token: token}
	}
}

func (m *OAuthCodex) stopPolling() tea.Msg {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	return nil
}
