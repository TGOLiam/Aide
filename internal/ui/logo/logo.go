// Package logo renders an Aide wordmark in a stylized way.
package logo

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/liamb/opencode/aide/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
)

// letterform represents a letterform. It can be stretched horizontally by
// a given amount via the boolean argument. Kept for compatibility with
// letterforms.go.
type letterform func(bool) string

const diag = `╱`

// Opts are the options for rendering the Aide title art.
type Opts struct {
	FieldColor   color.Color // diagonal lines (diagonal fill)
	TitleColorA  color.Color // ">_" prompt color (orange)
	TitleColorB  color.Color // "AIDE" wordmark color
	CharmColor   color.Color // Charm™ text color (unused in new design)
	VersionColor color.Color // version text color
	Width        int         // width of the rendered logo, used for truncation
	Hyper        bool        // whether it is Aide or Hyperaide

	Unstable bool
}

// Render renders the Aide logo.
//
// When compact=true, returns a 3-line sidebar logo block:
//
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	>_ AIDE
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//
// When compact=false, returns a wide version with diagonal side fields.
func Render(base lipgloss.Style, version string, compact bool, o Opts) string {
	fg := func(c color.Color, s string) string {
		return lipgloss.NewStyle().Foreground(c).Render(s)
	}

	name := "AIDE"
	if o.Hyper {
		name = "HYPERAIDE"
	}

	prompt := fg(o.TitleColorA, ">_ ")
	wordmark := fg(o.TitleColorB, name)

	if compact {
		return prompt + wordmark
	}

	// Wide version with diagonal side fields.
	content := prompt + wordmark
	versionStr := ""
	if version != "" {
		versionStr = "  " + fg(o.VersionColor, "v"+version)
	}
	metaRow := content + versionStr

	metaWidth := lipgloss.Width(metaRow)

	// Left field
	const leftWidth = 6
	leftFieldRow := fg(o.FieldColor, strings.Repeat(diag, leftWidth))

	// Right field - diagonal fill
	rightWidth := max(15, o.Width-leftWidth-metaWidth-4)
	rightFieldRow := fg(o.FieldColor, strings.Repeat(diag, rightWidth))

	logo := lipgloss.JoinHorizontal(lipgloss.Top, leftFieldRow, "  ", metaRow, "  ", rightFieldRow)

	if o.Width > 0 {
		logo = ansi.Truncate(logo, o.Width, "")
	}
	return logo
}

// SmallRender renders a small inline version of the Aide logo: ">_ aide"
// with diagonal fill to the right, suitable for sidebar header or topbar.
func SmallRender(t *styles.Styles, width int, o Opts) string {
	title := lipgloss.NewStyle().Foreground(o.TitleColorA).Render(">_ ")
	title += styles.ApplyBoldForegroundGrad(t.Logo.GradCanvas, "AIDE", t.Logo.SmallGradFromColor, t.Logo.SmallGradToColor)
	remainingWidth := width - lipgloss.Width(title) - 1
	if remainingWidth > 0 {
		lines := strings.Repeat("╱", remainingWidth)
		title = fmt.Sprintf("%s %s", title, t.Logo.SmallDiagonals.Render(lines))
	}
	return title
}

// LogoBlock returns a centered logo suitable for the sidebar logo area.
func LogoBlock(t *styles.Styles, width int) string {
	prompt := lipgloss.NewStyle().Foreground(t.Logo.TitleColorA).Render(">_ ")
	name := styles.ApplyBoldForegroundGrad(t.Logo.GradCanvas, "AIDE", t.Logo.TitleColorA, t.Logo.TitleColorB)
	line := prompt + name
	lineWidth := lipgloss.Width(line)
	leftPad := max(0, (width-lineWidth)/2)
	padding := strings.Repeat(" ", leftPad)
	return padding + line
}
