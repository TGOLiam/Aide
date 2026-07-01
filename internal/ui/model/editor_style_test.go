package model

import (
	"strings"
	"testing"
)

func TestBlurredTextareaKeepsBorder(t *testing.T) {
	t.Parallel()

	u := newTestUI()
	u.textarea.Blur()

	view := u.textarea.View()
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╰") {
		t.Fatalf("expected blurred textarea to retain border, got %q", view)
	}
}
