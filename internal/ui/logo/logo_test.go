package logo

import (
	"strings"
	"testing"

	"github.com/liamb/aide/internal/ui/styles"
)

func TestCompactRenderUsesSidebarBrandFormat(t *testing.T) {
	t.Parallel()

	s := styles.AideOrange()
	got := Render(s.Logo.GradCanvas, "", true, Opts{
		FieldColor:   s.Logo.FieldColor,
		TitleColorA:  s.Logo.TitleColorA,
		TitleColorB:  s.Logo.TitleColorB,
		VersionColor: s.Logo.VersionColor,
		Width:        30,
	})

	if !strings.Contains(got, "////") {
		t.Fatalf("expected compact logo to include left slashes, got %q", got)
	}
	if !strings.Contains(got, ">_ ") || !strings.Contains(got, "Aide") {
		t.Fatalf("expected compact logo to include >_ Aide, got %q", got)
	}
}
