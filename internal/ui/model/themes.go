package model

import (
	"github.com/liamb/aide/internal/ui/common"
	"github.com/liamb/aide/internal/ui/styles"
)

// cacheSidebarLogo renders and caches the sidebar logo at the specified width.
func (m *UI) cacheSidebarLogo(width int) {
	m.sidebarLogo = renderLogo(m.com.Styles, true, m.com.IsHyper(), width)
}

// applyThemeForProvider swaps the active theme to the one associated with
// the given provider, but only when that theme differs from the one
// already applied. Most providers share a single theme, so re-selecting a
// model from the same theme family would otherwise pay the full cost of
// invalidating the markdown renderer cache and re-rendering the entire
// transcript for no visible change.
func (m *UI) applyThemeForProvider(providerID string) {
	key := styles.ThemeKeyForProvider(providerID)
	if key == m.themeKey {
		return
	}
	m.themeKey = key
	m.applyTheme(styles.ThemeForProvider(providerID))
}

// applyTheme replaces the active styles with the given theme, drops the
// shared markdown renderer cache, and refreshes every component that
// caches style data.
func (m *UI) applyTheme(s styles.Styles) {
	*m.com.Styles = s
	common.InvalidateMarkdownRendererCache()
	m.refreshStyles()
}

// refreshStyles pushes the current *m.com.Styles into every subcomponent
// that copies or pre-renders style-dependent values at construction time.
func (m *UI) refreshStyles() {
	t := m.com.Styles
	m.header.refresh()
	if m.layout.sidebar.Dx() > 0 {
		m.cacheSidebarLogo(m.layout.sidebar.Dx())
	}
	m.textarea.SetStyles(t.Editor.Textarea)
	m.completions.SetStyles(t.Completions.Normal, t.Completions.Focused, t.Completions.Match)
	m.attachments.Renderer().SetStyles(
		t.Attachments.Normal,
		t.Attachments.Deleting,
		t.Attachments.Image,
		t.Attachments.Text,
		t.Attachments.Skill,
	)
	m.todoSpinner.Style = t.Pills.TodoSpinner
	m.status.help.Styles = t.Help
	m.chat.InvalidateRenderCaches()
}
