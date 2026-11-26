package templates

import (
	"strings"
	"testing"

	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
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

func TestRenderHomeCustomSections(t *testing.T) {
	settings := fullSettings()
	settings.SpotifyPlaylist = ""
	content := HomeContent{
		CustomSections: []models.CustomSection{
			{ID: 7, Position: 2, HeaderEn: "Extra", HeaderIt: "Extra", BodyEn: "More", BodyIt: "Altro", DesktopMediaID: 51, MobileMediaID: 52},
			{ID: 8, Position: 99, HeaderEn: "Last", HeaderIt: "Ultima", BodyEn: "End", BodyIt: "Fine"},
		},
	}
	out := render(t, Home(settings, content, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	for _, want := range []string{
		`id="custom-7"`,
		`id="custom-8"`,
		`<h2 class="section-title">Extra</h2>`,
		`<p class="custom-section-body">More</p>`,
		`srcset="/media/51"`,
		`vp-custom-section-with-background`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("custom sections render missing %q", want)
		}
	}
	// position 2 sits between landing and venues; position 99 sits last
	landing := strings.Index(out, `id="landing"`)
	custom := strings.Index(out, `id="custom-7"`)
	venues := strings.Index(out, `id="venues"`)
	last := strings.Index(out, `id="custom-8"`)
	if !(landing < custom && custom < venues && venues < last) {
		t.Errorf("custom order wrong: landing %d custom %d venues %d last %d", landing, custom, venues, last)
	}
	// the arrows navigate through the custom section
	if !strings.Contains(out, `<a href="#custom-7" class="vp-arrow vp-arrow-down"`) {
		t.Error("landing down arrow misses the custom section")
	}
	if !strings.Contains(out, `<a href="#landing" class="vp-arrow vp-arrow-up"`) {
		t.Error("custom section up arrow misses landing")
	}
}

func TestRenderHomeCustomSectionsLanguage(t *testing.T) {
	settings := fullSettings()
	settings.SpotifyPlaylist = ""
	content := HomeContent{
		CustomSections: []models.CustomSection{
			{ID: 9, Position: 2, HeaderIt: "Solo italiano", BodyIt: "Testo"},
		},
	}
	out := render(t, Home(settings, content, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if strings.Contains(out, `id="custom-9"`) {
		t.Error("italian-only section renders in English")
	}
	out = render(t, Home(settings, content, nil, nil, false, false, false, "", false, i18n.NewT("it"), "it", OGMeta{}))
	if !strings.Contains(out, `id="custom-9"`) || !strings.Contains(out, "Solo italiano") {
		t.Error("italian-only section missing in Italian")
	}
}

func TestRenderHomeCustomHoneymoon(t *testing.T) {
	settings := fullSettings()
	settings.SpotifyPlaylist = ""
	custom := models.CustomSection{ID: 10, Position: 5, HeaderEn: "Our honeymoon", HeaderIt: "La luna di miele", BodyEn: "The trip", BodyIt: "Il viaggio"}

	// no honeymoon places: the built-in section stays hidden, the custom stands alone
	content := HomeContent{CustomSections: []models.CustomSection{custom}}
	out := render(t, Home(settings, content, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, `id="custom-10"`) {
		t.Error("custom honeymoon section missing with no honeymoon places")
	}
	if strings.Contains(out, `id="honeymoon"`) {
		t.Error("built-in honeymoon renders with no honeymoon places")
	}

	// honeymoon places: both the custom and the built-in section render
	content.Honeymoon = []models.Place{{Label: "Safari", Name: "Serengeti"}}
	out = render(t, Home(settings, content, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, `id="custom-10"`) || !strings.Contains(out, `id="honeymoon"`) {
		t.Error("custom and built-in honeymoon do not both render")
	}
}
