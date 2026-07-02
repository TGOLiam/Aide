package model

import (
	"image"
	"math/rand"
	"os"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/liamb/aide/internal/version"
)

// drawHeader draws the header section of the UI.
func (m *UI) drawHeader(scr uv.Screen, area uv.Rectangle) {
	m.header.drawHeader(
		scr,
		area,
		m.session,
		m.isCompact,
		m.detailsOpen,
		area.Dx(),
	)
}

// Draw implements [uv.Drawable] and draws the UI model.
func (m *UI) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	layout := m.generateLayout(area.Dx(), area.Dy())

	if m.layout != layout {
		m.layout = layout
		m.updateSize()
	} else if m.state == uiChat && m.hasSession() {
		// Re-render pills on every draw so the box appears even when
		// the layout footprint hasn't changed (e.g. todos arrived
		// while the panel was collapsed). updateSize already calls
		// renderPills, but only when the layout actually differs;
		// this catches the steady-state case.
		m.renderPills()
	}

	// Clear the screen first
	screen.Clear(scr)

	switch m.state {
	case uiOnboarding:
		m.drawHeader(scr, layout.header)

		// NOTE: Onboarding flow will be rendered as dialogs below, but
		// positioned at the bottom left of the screen.

	case uiInitialize:
		m.drawHeader(scr, layout.header)

		main := uv.NewStyledString(m.initializeView())
		main.Draw(scr, layout.main)

	case uiLanding:
		m.drawHeader(scr, layout.header)
		main := uv.NewStyledString(m.landingView())
		main.Draw(scr, layout.main)

		editor := uv.NewStyledString(m.renderEditorView(scr.Bounds().Dx()))
		editor.Draw(scr, layout.editor)

	case uiChat:
		if m.isCompact {
			if m.useTopbarLayout {
				drawTopbar(scr, layout.header, m.com, m.session, m.width)
			} else {
				m.drawHeader(scr, layout.header)
			}
		} else {
			m.drawSidebar(scr, layout.sidebar)
		}

		m.chat.Draw(scr, layout.main)
		if layout.pills.Dy() > 0 && m.pillsView != "" {
			uv.NewStyledString(m.pillsView).Draw(scr, layout.pills)
		}

		editorWidth := scr.Bounds().Dx()
		if !m.isCompact {
			editorWidth -= layout.sidebar.Dx()
		}
		editor := uv.NewStyledString(m.renderEditorView(editorWidth))
		editor.Draw(scr, layout.editor)

		// Draw green accent border in compact topbar mode (Layout 3)
		if m.isCompact && m.useTopbarLayout {
			borderStyle := uv.NewStyledString(
				m.com.Styles.AccentBorder.Render("▎"),
			)
			borderStyle.Draw(scr, image.Rectangle{
				Min: image.Pt(layout.main.Min.X-1, layout.main.Min.Y),
				Max: image.Pt(layout.main.Min.X, layout.editor.Max.Y),
			})
		}

		// Draw details overlay in compact mode when open
		if m.isCompact && m.detailsOpen {
			m.drawSessionDetails(scr, layout.sessionDetails)
		}
	}

	isOnboarding := m.state == uiOnboarding

	// Add status and help layer
	m.status.SetHideHelp(isOnboarding)
	m.status.Draw(scr, layout.status)

	// Draw completions popup if open
	if !isOnboarding && m.completionsOpen && m.completions.HasItems() {
		w, h := m.completions.Size()
		x := m.completionsPositionStart.X
		y := m.completionsPositionStart.Y - h

		screenW := area.Dx()
		if x+w > screenW {
			x = screenW - w
		}
		x = max(0, x)
		y = max(0, y+1) // Offset for attachments row

		completionsView := uv.NewStyledString(m.completions.Render())
		completionsView.Draw(scr, image.Rectangle{
			Min: image.Pt(x, y),
			Max: image.Pt(x+w, y+h),
		})
	}

	// Debugging rendering (visually see when the tui rerenders)
	if os.Getenv("AIDE_UI_DEBUG") == "true" {
		debugView := lipgloss.NewStyle().Background(lipgloss.ANSIColor(rand.Intn(256))).Width(4).Height(2)
		debug := uv.NewStyledString(debugView.String())
		debug.Draw(scr, image.Rectangle{
			Min: image.Pt(4, 1),
			Max: image.Pt(8, 3),
		})
	}

	// This needs to come last to overlay on top of everything. We always pass
	// the full screen bounds because the dialogs will position themselves
	// accordingly.
	if m.dialog.HasDialogs() {
		return m.dialog.Draw(scr, scr.Bounds())
	}

	switch m.focus {
	case uiFocusEditor:
		if m.layout.editor.Dy() <= 0 {
			// Don't show cursor if editor is not visible
			return nil
		}
		if m.detailsOpen && m.isCompact {
			// Don't show cursor if details overlay is open
			return nil
		}

		if m.textarea.Focused() {
			cur := m.textarea.Cursor()
			cur.X++ // Adjust for app margins.
			cur.Y += m.layout.editor.Min.Y + 1 // Offset for attachments row.
			return cur
		}
	}
	return nil
}

// drawSessionDetails draws the session details in compact mode.
func (m *UI) drawSessionDetails(scr uv.Screen, area uv.Rectangle) {
	if m.session == nil {
		return
	}

	s := m.com.Styles

	width := area.Dx() - s.CompactDetails.View.GetHorizontalFrameSize()
	height := area.Dy() - s.CompactDetails.View.GetVerticalFrameSize()

	title := s.CompactDetails.Title.Width(width).MaxHeight(2).Render(m.session.Title)
	blocks := []string{
		title,
		"",
		m.modelInfo(width),
		"",
	}

	detailsHeader := lipgloss.JoinVertical(
		lipgloss.Left,
		blocks...,
	)

	ver := s.CompactDetails.Version.Width(width).AlignHorizontal(lipgloss.Right).Render(version.Version)

	remainingHeight := height - lipgloss.Height(detailsHeader) - lipgloss.Height(ver)

	const maxSectionWidth = 50
	sectionWidth := max(1, min(maxSectionWidth, width/4-2)) // account for spacing between sections
	maxItemsPerSection := remainingHeight - 3               // Account for section title and spacing

	lspSection := m.lspInfo(sectionWidth, maxItemsPerSection, false)
	mcpSection := m.mcpInfo(sectionWidth, maxItemsPerSection, false)
	skillsSection := m.skillsInfo(sectionWidth, maxItemsPerSection, false)
	filesSection := m.filesInfo(m.com.Workspace.WorkingDir(), sectionWidth, maxItemsPerSection, false)
	sections := lipgloss.JoinHorizontal(lipgloss.Top, filesSection, " ", lspSection, " ", mcpSection, " ", skillsSection)
	uv.NewStyledString(
		s.CompactDetails.View.
			Width(area.Dx()).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					detailsHeader,
					sections,
					ver,
				),
			),
	).Draw(scr, area)
}
