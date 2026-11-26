package templates

import (
	"strings"
	"testing"

	"github.com/streambinder/foedus/internal/i18n"
)

func TestHeroAnchorClass(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"top", "vp-hero-anchor-top"},
		{"center", "vp-hero-anchor-center"},
		{"bottom", "vp-hero-anchor-bottom"},
		{"", "vp-hero-anchor-bottom"},
		{"sideways", "vp-hero-anchor-bottom"},
	} {
		if got := heroAnchorClass(tc.in); got != tc.want {
			t.Errorf("heroAnchorClass(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderHomeHeroAnchor(t *testing.T) {
	for _, tc := range []struct {
		anchor string
		want   string
		absent string
	}{
		{"top", "vp-hero-anchor-top", "vp-hero-anchor-bottom"},
		{"center", "vp-hero-anchor-center", "vp-hero-anchor-bottom"},
		{"bottom", "vp-hero-anchor-bottom", "vp-hero-anchor-top"},
		{"", "vp-hero-anchor-bottom", "vp-hero-anchor-top"},
	} {
		t.Run(tc.anchor, func(t *testing.T) {
			settings := fullSettings()
			settings.HeroTextAnchor = tc.anchor
			out := render(t, Home(settings, HomeContent{}, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
			if !strings.Contains(out, tc.want) {
				t.Errorf("anchor %q renders without %q", tc.anchor, tc.want)
			}
			if strings.Contains(out, tc.absent) {
				t.Errorf("anchor %q renders with %q", tc.anchor, tc.absent)
			}
		})
	}
	// the anchor class rides on the landing section itself
	settings := fullSettings()
	settings.HeroTextAnchor = "top"
	out := render(t, Home(settings, HomeContent{}, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, `<section id="landing" class="vp-section vp-hero vp-hero-anchor-top"`) {
		t.Errorf("landing section misses the anchor class: %.300s", out)
	}
}

func TestRenderHomeGiftsCoordinateMode(t *testing.T) {
	settings, content, items, claimed := homeFixtures()
	settings.GiftsMode = "coordinate"

	out := render(t, Home(settings, content, items, claimed, true, false, false, "", false, i18n.NewT("it"), "it", OGMeta{}))
	// the single box is a gift tile like the wedding-list ones: the gift
	// icon above the text, IBAN in code, beneficiary below it, Copia below
	// that, styled like the Dona button
	for _, want := range []string{
		`id="coordinate-iban"`, "IT60X0542811101000000123456",
		`class="gift-coordinate-owner"`, "Davide Pucci",
		`data-target="coordinate-iban"`, ">Copia</button>",
		`<h2 class="section-title">Regalo</h2>`, "Nessuna idea? Ci pensiamo noi.",
		`class="gift-tile"`, `class="free-gift-image-placeholder"`,
		`class="gift-tile-btn copy-btn"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("coordinate mode missing %q", want)
		}
	}
	if icon := strings.Index(out, `class="free-gift-image-placeholder"`); icon < 0 ||
		strings.Index(out, `id="coordinate-iban"`) < icon {
		t.Error("coordinate box must show the gift icon above the IBAN")
	}
	if iban := strings.Index(out, `id="coordinate-iban"`); iban < 0 ||
		strings.Index(out, `class="gift-coordinate-owner"`) < iban ||
		strings.Index(out, ">Copia</button>") < strings.Index(out, `class="gift-coordinate-owner"`) {
		t.Error("coordinate box must order IBAN, beneficiary, then Copia")
	}
	// no registry item, no Dona button, no claim modal; the section title
	// is the coordinate one (the venues arrow keeps "Lista nozze" as the
	// aria-label of the link that points at this section)
	for _, absent := range []string{
		`class="gift-tile gift-tile-is-gifted"`,
		"Toaster", "Lamp", "Vase", "Open gift", ">Dona<",
		`id="transfer-modal"`, `id="pct-modal"`, `id="thanks-modal"`,
		`<h2 class="section-title">Lista nozze</h2>`,
	} {
		if strings.Contains(out, absent) {
			t.Errorf("coordinate mode renders %q", absent)
		}
	}

	// the english title defaults to Gift and the button to Copy
	out = render(t, Home(settings, content, items, claimed, true, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	for _, want := range []string{"Gift", ">Copy</button>"} {
		if !strings.Contains(out, want) {
			t.Errorf("coordinate mode in english missing %q", want)
		}
	}
	if strings.Contains(out, `<h2 class="section-title">Wedding List</h2>`) {
		t.Error("coordinate mode in english renders the standard title")
	}
}

func TestRenderHomeGiftsStandardMode(t *testing.T) {
	settings, content, items, claimed := homeFixtures()
	settings.GiftsMode = "standard"

	out := render(t, Home(settings, content, items, claimed, true, false, false, "", false, i18n.NewT("it"), "it", OGMeta{}))
	for _, want := range []string{
		`class="gift-tile"`, "Toaster", ">Dona<",
		`id="transfer-modal"`, `id="pct-modal"`, `id="thanks-modal"`,
		"Lista nozze",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("standard mode missing %q", want)
		}
	}
	if strings.Contains(out, `id="coordinate-iban"`) {
		t.Error("standard mode renders the coordinate box")
	}
}
