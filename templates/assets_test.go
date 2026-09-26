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
