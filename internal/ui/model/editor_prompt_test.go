package model

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
)

func TestNormalPromptFuncRendersGrayEmptyStateHint(t *testing.T) {
	t.Parallel()

	u := newTestUI()
	u.readyPlaceholder = "Ready!"

	got := u.normalPromptFunc(textarea.PromptInfo{
		LineNumber: 0,
		Focused:    true,
	})

	if !strings.Contains(got, "Ready!") {
		t.Fatalf("expected prompt to include placeholder text, got %q", got)
	}
	if !strings.Contains(got, ":::") {
		t.Fatalf("expected prompt to use ::: prefix, got %q", got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected prompt to include styled placeholder text, got %q", got)
	}
}

func TestNormalPromptFuncUsesConsistentFirstLinePrefix(t *testing.T) {
	t.Parallel()

	u := newTestUI()
	u.textarea.SetValue("hello")

	got := u.normalPromptFunc(textarea.PromptInfo{
		LineNumber: 0,
		Focused:    true,
	})

	if !strings.Contains(got, ":::") {
		t.Fatalf("expected first line prompt to use :::, got %q", got)
	}
	if strings.Contains(got, ">") {
		t.Fatalf("expected first line prompt to omit >, got %q", got)
	}
}
