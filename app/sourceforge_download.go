package main

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

// SourceForge's HTTP-200 download page redirects using HTML, not a Location
// header. Follow only its exact project/file on the official download host.
func sourceForgeDownloadRedirect(original string, page []byte) string {
	from, err := url.Parse(original)
	if err != nil || from.Scheme != "https" || from.Host != "sourceforge.net" {
		return ""
	}
	parts := strings.SplitN(strings.TrimPrefix(from.Path, "/projects/"), "/files/", 2)
	if !strings.HasPrefix(from.Path, "/projects/") || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	want := "/project/" + parts[0] + "/" + strings.TrimSuffix(parts[1], "/download")
	match := regexp.MustCompile(`(?i)<meta\s+http-equiv=["']refresh["']\s+content=["'][0-9]+;\s*url=([^"']+)`).FindSubmatch(page)
	if len(match) != 2 {
		return ""
	}
	target, err := url.Parse(html.UnescapeString(string(match[1])))
	if err != nil || target.Scheme != "https" || target.Host != "downloads.sourceforge.net" || target.User != nil || target.Path != want {
		return ""
	}
	return target.String()
}
