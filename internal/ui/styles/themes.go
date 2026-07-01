package styles

// ThemeKeyForProvider returns a stable identifier for the theme
// associated with the given provider ID.
func ThemeKeyForProvider(providerID string) string {
	return "aide"
}

// ThemeForProvider returns the Styles associated with the given provider
// ID. Unknown or empty provider IDs yield the Aide Orange theme.
func ThemeForProvider(providerID string) Styles {
	return AideOrange()
}

// AideOrange returns the Aide orange theme.
func AideOrange() Styles {
	s := quickStyle(quickStyleOpts{
		primary:   rgba("#c85f1e"), // Orange — primary accent
		secondary: rgba("#e07840"), // Orange light
		accent:    rgba("#8a4010"), // Orange dim
		keyword:   rgba("#4a2008"), // Orange faint

		fgBase:       rgba("#deded6"), // text-primary
		fgMoreSubtle: rgba("#a8a8a0"), // text-secondary
		fgSubtle:     rgba("#a3a39c"), // text-dim (lightened for readability)
		fgMostSubtle: rgba("#686864"), // text-faint (lightened)

		onPrimary: rgba("#deded6"), // text on orange bg

		bgBase:         rgba("#0c0e0d"), // bg-terminal
		bgLeastVisible: rgba("#0f1110"), // bg-chrome
		bgLessVisible:  rgba("#09100a"), // bg-bar
		bgMostVisible:  rgba("#1c1e1c"), // border-chrome

		separator: rgba("#1c1e1c"), // border-chrome

		destructive:       rgba("#b85858"), // text-diff-rem
		error:             rgba("#b85858"),
		warningSubtle:     rgba("#8a8a84"),
		warning:           rgba("#e07840"),
		denied:            rgba("#b85858"),
		busy:              rgba("#8a8a84"),
		info:              rgba("#a8a8a0"),
		infoMoreSubtle:    rgba("#6e6e68"), // text-muted (lightened)
		infoMostSubtle:    rgba("#686864"),
		success:           rgba("#00FFB2"), // text-shell-out (Julep)
		successMoreSubtle: rgba("#68FFD6"), // text-diff-add (Bok)
		successMostSubtle: rgba("#12C78F"), // bg-diff-add (Guac)

		// ANSI 16-color palette
		ansiBlack:        rgba("#0c0e0d"),
		ansiRed:          rgba("#b85858"),
		ansiGreen:        rgba("#00FFB2"),
		ansiYellow:       rgba("#e07840"),
		ansiBlue:         rgba("#6e6e68"),
		ansiMagenta:      rgba("#a3a39c"),
		ansiCyan:         rgba("#a8a8a0"),
		ansiWhite:        rgba("#deded6"),
		ansiBrightBlack:  rgba("#686864"),
		ansiBrightRed:    rgba("#b85858"),
		ansiBrightGreen:  rgba("#68FFD6"),
		ansiBrightYellow: rgba("#c85f1e"),
		ansiBrightBlue:   rgba("#6e6e68"),
		ansiBrightMagenta: rgba("#8a8a84"),
		ansiBrightCyan:   rgba("#a8a8a0"),
		ansiBrightWhite:  rgba("#deded6"),
	})

	return s
}

// CharmtonePantera returns the Aide Orange theme (default alias).
func CharmtonePantera() Styles {
	return AideOrange()
}

// HypercrushObsidiana returns the Hypercrush dark theme (alias, kept for compatibility).
func HypercrushObsidiana() Styles {
	return AideOrange()
}
