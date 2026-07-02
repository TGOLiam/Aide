package dialog

import (
	tea "charm.land/bubbletea/v2"
	"github.com/liamb/aide/internal/ui/common"
)

// CustomProviderID is the identifier for the custom provider dialog.
const CustomProviderID = "custom-provider"

// ActionOpenCustomProviderDialog opens the custom provider dialog.
type ActionOpenCustomProviderDialog struct {
	ModelType ModelType
}

// CustomProviderInput is a compatibility wrapper around the model selector.
type CustomProviderInput struct {
	*Models
}

var _ Dialog = (*CustomProviderInput)(nil)

// NewCustomProviderInput creates a custom provider dialog.
func NewCustomProviderInput(com *common.Common, modelType ModelType) *CustomProviderInput {
	m, err := NewModels(com, false)
	if err != nil {
		return &CustomProviderInput{Models: &Models{com: com, modelType: modelType}}
	}
	m.modelType = modelType
	_ = m.setProviderItems()
	return &CustomProviderInput{Models: m}
}

// ID implements Dialog.
func (m *CustomProviderInput) ID() string {
	return CustomProviderID
}

// HandleMsg implements Dialog.
func (m *CustomProviderInput) HandleMsg(msg tea.Msg) Action {
	if m.Models == nil {
		return nil
	}
	return m.Models.HandleMsg(msg)
}
