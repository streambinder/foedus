package templates

import (
	"net/url"
	"strings"

	"github.com/streambinder/foedus/internal/buildinfo"
)

func assetURL(path string) string {
	path = strings.TrimSpace(path)
	version := strings.TrimSpace(buildinfo.AssetVersion)
	if path == "" || version == "" || !strings.HasPrefix(path, "/static/") {
		return path
	}

	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "v=" + url.QueryEscape(version)
}
