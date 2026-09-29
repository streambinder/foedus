package templates

import (
	"strings"
	"testing"

	"github.com/streambinder/foedus/internal/models"
)

// appearanceVars lands verbatim in a style attribute on the public invitation
// page, so this pins both the pass-through and the invalid-input fallback.
func TestAppearanceVars(t *testing.T) {
	got := appearanceVars(models.Settings{EnvelopeColor: "#123abc", PageBgColor: "#f0e6d8"})
	if want := "--envelope-base:#123abc;--invitation-page-bg:#f0e6d8;"; got != want {
		t.Errorf("appearanceVars() = %q, want %q", got, want)
	}

	// a hand-edited row can never inject into the style attribute
	got = appearanceVars(models.Settings{EnvelopeColor: `red";background:url(evil)`, PageBgColor: ""})
	if want := "--envelope-base:" + models.DefaultEnvelopeColor + ";--invitation-page-bg:" + models.DefaultPageBgColor + ";"; got != want {
		t.Errorf("appearanceVars() with invalid input = %q, want %q", got, want)
	}
	if strings.ContainsAny(got, `"()`) {
		t.Errorf("appearanceVars() output %q contains injectable characters", got)
	}
}

// homePageStyle lands verbatim in the style attribute of <html> on the
// homepage, so this pins both the pass-through and the invalid-input
// fallback.
func TestHomePageStyle(t *testing.T) {
	got := homePageStyle(models.Settings{HomeBgColor: "#e8f0e0"})
	if want := "--home-page-bg:#e8f0e0;"; got != want {
		t.Errorf("homePageStyle() = %q, want %q", got, want)
	}

	// a hand-edited row can never inject into the style attribute
	got = homePageStyle(models.Settings{HomeBgColor: `red";background:url(evil)`})
	if want := "--home-page-bg:" + models.DefaultHomeBgColor + ";"; got != want {
		t.Errorf("homePageStyle() with invalid input = %q, want %q", got, want)
	}
	if strings.ContainsAny(got, `"()`) {
		t.Errorf("homePageStyle() output %q contains injectable characters", got)
	}
}
