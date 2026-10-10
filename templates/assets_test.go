package templates

import (
	"testing"

	"github.com/streambinder/foedus/internal/buildinfo"
)

func TestAssetURLAppendsBuildVersionToStaticAssets(t *testing.T) {
	previous := buildinfo.AssetVersion
	buildinfo.AssetVersion = "2026 05 05"
	t.Cleanup(func() {
		buildinfo.AssetVersion = previous
	})

	got := assetURL("/static/css/styles.css")
	want := "/static/css/styles.css?v=2026+05+05"
	if got != want {
		t.Fatalf("assetURL() = %q, want %q", got, want)
	}
}

func TestAssetURLUsesAmpersandWhenQueryPresent(t *testing.T) {
	previous := buildinfo.AssetVersion
	buildinfo.AssetVersion = "v1"
	t.Cleanup(func() {
		buildinfo.AssetVersion = previous
	})

	got := assetURL("/static/app.css?lang=en")
	if want := "/static/app.css?lang=en&v=v1"; got != want {
		t.Fatalf("assetURL() = %q, want %q", got, want)
	}
	// without a build version nothing is appended, query or not
	buildinfo.AssetVersion = ""
	if got := assetURL("/static/app.css?lang=en"); got != "/static/app.css?lang=en" {
		t.Fatalf("assetURL() without version = %q", got)
	}
	// whitespace is trimmed before the static prefix is checked
	buildinfo.AssetVersion = "v1"
	if got := assetURL("  /static/app.css  "); got != "/static/app.css?v=v1" {
		t.Fatalf("assetURL() with padded path = %q", got)
	}
}

func TestAssetURLLeavesNonStaticAssetsUnchanged(t *testing.T) {
	previous := buildinfo.AssetVersion
	buildinfo.AssetVersion = "20260505"
	t.Cleanup(func() {
		buildinfo.AssetVersion = previous
	})

	got := assetURL("/media/example")
	if got != "/media/example" {
		t.Fatalf("assetURL() = %q, want /media/example", got)
	}
}
