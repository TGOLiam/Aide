package model

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/liamb/aide/internal/config"
	"github.com/liamb/aide/internal/fsext"
	"github.com/liamb/aide/internal/session"
	"github.com/liamb/aide/internal/ui/common"
	"github.com/liamb/aide/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

const (
	headerDiag           = "╱"
	minHeaderDiags       = 3
	leftPadding          = 1
	rightPadding         = 1
	diagToDetailsSpacing = 1 // space between diagonal pattern and details section
)

type header struct {
	// cached logo and compact logo
	logo        string
	compactLogo string

	com     *common.Common
	width   int
	compact bool
}

// newHeader creates a new header model.
func newHeader(com *common.Common) *header {
	h := &header{
		com: com,
	}
	h.refresh()
	return h
}

// refresh rebuilds cached logo strings using the current styles. Call
// after the theme changes.
func (h *header) refresh() {
	t := h.com.Styles
	isHyper := h.com.IsHyper()
	charm := "Charm™"
	if !isHyper {
		charm = " " + charm
	}
	name := "AIDE"
	if isHyper {
		name = "HYPERAIDE"
	}
	h.compactLogo = t.Header.Charm.Render(charm) + " " +
		styles.ApplyBoldForegroundGrad(t.Header.LogoGradCanvas, name, t.Header.LogoGradFromColor, t.Header.LogoGradToColor) + " "
	// Force drawHeader to re-render the wide logo on the next frame.
	h.width = 0
	h.logo = ""
}

// drawTopbar renders the Layout 3 compact topbar.
//
//	>_ aide /////////////////// ~/path • 13% • ctrl+d open
func drawTopbar(
	scr uv.Screen,
	area uv.Rectangle,
	com *common.Common,
	session *session.Session,
	width int,
) {
	t := com.Styles

	// Logo part: ">_ AIDE"
	logo := lipgloss.NewStyle().Foreground(t.Logo.TitleColorA).Render(">_ ") +
		lipgloss.NewStyle().Foreground(t.Logo.SmallGradToColor).Render("AIDE")

	// Working directory
	dirTrimLimit := 4
	cwd := fsext.DirTrim(fsext.PrettyPath(com.Workspace.WorkingDir()), dirTrimLimit)
	cwdStr := t.Topbar.WorkingDir.Render(cwd)

	// Context percentage
	percentStr := ""
	if session != nil {
		agentCfg := com.Config().Agents[config.AgentCoder]
		model := com.Config().GetModelByType(agentCfg.Model)
		if model != nil && model.ContextWindow > 0 {
			percentage := (float64(session.CompletionTokens+session.PromptTokens) / float64(model.ContextWindow)) * 100
			percentageText := fmt.Sprintf("%d%%", int(percentage))
			if session.EstimatedUsage {
				percentageText = "~" + percentageText
			}
			percentStr = t.Topbar.Percentage.Render(percentageText)
		}
	}

	// Keystroke hint
	keystroke := t.Topbar.Keystroke.Render("ctrl+d") + t.Topbar.KeystrokeTip.Render(" open")

	// Build the right-side details
	var detailParts []string
	if percentStr != "" {
		detailParts = append(detailParts, percentStr)
	}
	detailParts = append(detailParts, keystroke)
	dot := t.Topbar.Separator.Render(" • ")
	details := strings.Join(detailParts, " "+dot+" ")

	// Calculate space for diagonals
	leftContent := logo
	rightContent := cwdStr
	if details != "" {
		rightContent = cwdStr + " " + dot + " " + details
	}

	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	remainingWidth := width - leftWidth - rightWidth - 4 // 4 for padding

	diagStr := ""
	if remainingWidth > 0 {
		diagStr = t.Topbar.Diagonals.Render(strings.Repeat("╱", remainingWidth))
	}

	// Combine
	var b strings.Builder
	b.WriteString(leftContent)
	if diagStr != "" {
		b.WriteString(" ")
		b.WriteString(diagStr)
		b.WriteString(" ")
	}
	b.WriteString(rightContent)

	view := uv.NewStyledString(
		t.Topbar.Wrapper.Render(b.String()),
	)
	view.Draw(scr, area)
}

// drawHeader draws the header for the given session.
func (h *header) drawHeader(
	scr uv.Screen,
	area uv.Rectangle,
	session *session.Session,
	compact bool,
	detailsOpen bool,
	width int,
) {
	t := h.com.Styles
	if width != h.width || compact != h.compact {
		h.logo = renderLogo(h.com.Styles, compact, h.com.IsHyper(), width)
	}

	h.width = width
	h.compact = compact

	if !compact || session == nil {
		uv.NewStyledString(h.logo).Draw(scr, area)
		return
	}

	if session.ID == "" {
		return
	}

	var b strings.Builder
	b.WriteString(h.compactLogo)

	availDetailWidth := width - leftPadding - rightPadding - lipgloss.Width(b.String()) - minHeaderDiags - diagToDetailsSpacing
	lspErrorCount := 0
	for _, info := range h.com.Workspace.LSPGetStates() {
		lspErrorCount += info.DiagnosticCount
	}
	details := renderHeaderDetails(
		h.com,
		session,
		lspErrorCount,
		detailsOpen,
		availDetailWidth,
	)

	remainingWidth := width -
		lipgloss.Width(b.String()) -
		lipgloss.Width(details) -
		leftPadding -
		rightPadding -
		diagToDetailsSpacing

	if remainingWidth > 0 {
		b.WriteString(t.Header.Diagonals.Render(
			strings.Repeat(headerDiag, max(minHeaderDiags, remainingWidth)),
		))
		b.WriteString(" ")
	}

	b.WriteString(details)

	view := uv.NewStyledString(
		t.Header.Wrapper.Padding(0, rightPadding, 0, leftPadding).Render(b.String()),
	)
	view.Draw(scr, area)
}

// renderHeaderDetails renders the details section of the header.
func renderHeaderDetails(
	com *common.Common,
	session *session.Session,
	lspErrorCount int,
	detailsOpen bool,
	availWidth int,
) string {
	t := com.Styles

	var parts []string

	if lspErrorCount > 0 {
		parts = append(parts, t.LSP.ErrorDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPErrorIcon, lspErrorCount)))
	}

	agentCfg := com.Config().Agents[config.AgentCoder]
	model := com.Config().GetModelByType(agentCfg.Model)
	if model != nil && model.ContextWindow > 0 {
		percentage := (float64(session.CompletionTokens+session.PromptTokens) / float64(model.ContextWindow)) * 100
		percentageText := fmt.Sprintf("%d%%", int(percentage))
		if session.EstimatedUsage {
			percentageText = "~" + percentageText
		}
		formattedPercentage := t.Header.Percentage.Render(percentageText)
		parts = append(parts, formattedPercentage)
	}

	const keystroke = "ctrl+d"
	if detailsOpen {
		parts = append(parts, t.Header.Keystroke.Render(keystroke)+t.Header.KeystrokeTip.Render(" close"))
	} else {
		parts = append(parts, t.Header.Keystroke.Render(keystroke)+t.Header.KeystrokeTip.Render(" open "))
	}

	dot := t.Header.Separator.Render(" • ")
	metadata := strings.Join(parts, dot)
	metadata = dot + metadata

	const dirTrimLimit = 4
	cwd := fsext.DirTrim(fsext.PrettyPath(com.Workspace.WorkingDir()), dirTrimLimit)
	cwd = t.Header.WorkingDir.Render(cwd)

	result := cwd + metadata
	return ansi.Truncate(result, max(0, availWidth), "…")
}
