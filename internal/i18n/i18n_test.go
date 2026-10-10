package i18n

import (
	"testing"
	"time"
)

func TestNewT(t *testing.T) {
	if got := NewT("en")("title.setup"); got != "Setup Required" {
		t.Errorf("english lookup = %q, want %q", got, "Setup Required")
	}
	if got := NewT("it")("title.setup"); got != "Configurazione necessaria" {
		t.Errorf("italian lookup = %q, want %q", got, "Configurazione necessaria")
	}
	// an unknown language falls back to the english map
	if got := NewT("de")("title.setup"); got != "Setup Required" {
		t.Errorf("unknown language lookup = %q, want english fallback", got)
	}
	// an unknown key falls back to the key itself
	if got := NewT("en")("no.such.key"); got != "no.such.key" {
		t.Errorf("unknown key lookup = %q, want the key itself", got)
	}
	// every homepage key must resolve in both supported languages
	for _, lang := range Languages {
		translator := NewT(lang)
		for _, key := range HomepageKeys {
			if got := translator(key); got == key {
				t.Errorf("language %s has no translation for homepage key %q", lang, key)
			}
		}
	}
}

func TestNewTWithOverrides(t *testing.T) {
	translator := NewTWithOverrides("en", map[string]string{"title.setup": "Custom Setup"})
	if got := translator("title.setup"); got != "Custom Setup" {
		t.Errorf("override lookup = %q, want %q", got, "Custom Setup")
	}
	// an empty override value falls through to the compiled-in default
	translator = NewTWithOverrides("en", map[string]string{"title.setup": ""})
	if got := translator("title.setup"); got != "Setup Required" {
		t.Errorf("empty override lookup = %q, want the default", got)
	}
	// a nil overrides map behaves like NewT
	translator = NewTWithOverrides("it", nil)
	if got := translator("title.setup"); got != "Configurazione necessaria" {
		t.Errorf("nil overrides lookup = %q, want %q", got, "Configurazione necessaria")
	}
	if got := translator("no.such.key"); got != "no.such.key" {
		t.Errorf("unknown key lookup = %q, want the key itself", got)
	}
}

func TestDefaults(t *testing.T) {
	for _, lang := range Languages {
		defaults := Defaults(lang)
		if len(defaults) != len(HomepageKeys) {
			t.Fatalf("Defaults(%q) returned %d entries, want %d", lang, len(defaults), len(HomepageKeys))
		}
		translator := NewT(lang)
		for _, key := range HomepageKeys {
			// some descriptions default to empty on purpose; the contract is
			// that Defaults mirrors the translator for every homepage key.
			if defaults[key] != translator(key) {
				t.Errorf("Defaults(%q)[%q] = %q, want translator value %q", lang, key, defaults[key], translator(key))
			}
		}
	}
	if got := Defaults("en")["home.ceremony"]; got != "The Ceremony" {
		t.Errorf("english default for home.ceremony = %q", got)
	}
	if got := Defaults("it")["home.ceremony"]; got != "Il Rito" {
		t.Errorf("italian default for home.ceremony = %q", got)
	}
}

func TestDetectLang(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{"empty header", "", "en"},
		{"english", "en", "en"},
		{"italian", "it", "it"},
		{"italian with region", "it-IT,it;q=0.9,en;q=0.8", "it"},
		{"english with region and quality", "en-US;q=0.9", "en"},
		{"unsupported first then italian", "fr-FR, it;q=0.5", "it"},
		{"unsupported only", "fr-FR, de;q=0.5", "en"},
		{"uppercase tag", "IT", "it"},
		{"single character part skipped", "i, en", "en"},
		{"blank parts", " , ", "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectLang(tc.header); got != tc.want {
				t.Errorf("DetectLang(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	date := time.Date(2026, 7, 17, 15, 30, 0, 0, time.UTC)
	if got := FormatDate(date, "en"); got != "Jul 17, 2026" {
		t.Errorf("english date = %q", got)
	}
	if got := FormatDate(date, "it"); got != "17 luglio 2026" {
		t.Errorf("italian date = %q", got)
	}
	if got := FormatDate(date, "de"); got != "Jul 17, 2026" {
		t.Errorf("other language date = %q, want english layout", got)
	}
}

func TestFormatDatetimeUniversal(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"2026-07-17T15:30", "17/07/2026 15:30"},
		{"2026-07-17", "17/07/2026"},
		{"not a date", "not a date"},
		{"", ""},
	} {
		if got := FormatDatetimeUniversal(tc.in); got != tc.want {
			t.Errorf("FormatDatetimeUniversal(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatDatetime(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		lang string
		want string
	}{
		{"english with time", "2026-07-17T15:30", "en", "July 17, 2026 at 3:30 PM"},
		{"english date only", "2026-07-17", "en", "July 17, 2026"},
		{"italian with time", "2026-07-17T15:30", "it", "17 luglio 2026, 15:30"},
		{"italian date only", "2026-07-17", "it", "17 luglio 2026"},
		{"invalid returns input", "garbage", "en", "garbage"},
		{"invalid italian returns input", "garbage", "it", "garbage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatDatetime(tc.in, tc.lang); got != tc.want {
				t.Errorf("FormatDatetime(%q, %q) = %q, want %q", tc.in, tc.lang, got, tc.want)
			}
		})
	}
}

func TestFormatDatetimeLines(t *testing.T) {
	if got := FormatDatetimeDateLine("2026-07-17T15:30", "en"); got != "Jul 17, 2026" {
		t.Errorf("english date line = %q", got)
	}
	if got := FormatDatetimeDateLine("2026-07-17T15:30", "it"); got != "17 luglio 2026" {
		t.Errorf("italian date line = %q", got)
	}
	if got := FormatDatetimeDateLine("garbage", "en"); got != "garbage" {
		t.Errorf("invalid date line = %q, want input", got)
	}
	if got := FormatDatetimeTimeLine("2026-07-17T15:30", "en"); got != "3:30 PM" {
		t.Errorf("english time line = %q", got)
	}
	if got := FormatDatetimeTimeLine("2026-07-17T15:30", "it"); got != "15:30" {
		t.Errorf("italian time line = %q", got)
	}
	if got := FormatDatetimeTimeLine("2026-07-17", "en"); got != "" {
		t.Errorf("date-only time line = %q, want empty", got)
	}
	if got := FormatDatetimeTimeLine("garbage", "en"); got != "" {
		t.Errorf("invalid time line = %q, want empty", got)
	}
}
