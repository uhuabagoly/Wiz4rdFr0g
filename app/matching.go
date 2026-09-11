package main

import (
	"regexp"
	"strings"
	"unicode"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

type searchRow struct {
	Name string
	ID   string
}

var registryDecorations = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\s*\((?:x64|x86|64[- ]?bit|32[- ]?bit)(?:\s+[^)]*)?\)\s*`),
	regexp.MustCompile(`(?i)\s*\([a-z]{2}(?:-[a-z]{2})?\)\s*`),
	regexp.MustCompile(`(?i)\s+(?:x64|x86|64[- ]?bit|32[- ]?bit)\s*$`),
	regexp.MustCompile(`(?i)\s+v?\d+(?:\.\d+){1,4}\s*$`),
}

func normalizeSearch(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func canonicalRegistryName(s string) string {
	s = strings.TrimSpace(s)
	for _, re := range registryDecorations {
		s = re.ReplaceAllString(s, " ")
	}
	return normalizeSearch(strings.Join(strings.Fields(s), " "))
}

func candidateQueries(name string) []string {
	return catalogpkg.CandidateQueries(name)
}

func catalogIDs(app appDef) []string {
	return catalogpkg.IDsFor(app)
}

func registryNamesCompatible(want, got string) bool {
	w := strings.ToLower(want)
	g := strings.ToLower(got)
	for _, token := range catalogpkg.RegistryVariantTokens {
		if !strings.Contains(w, token) && strings.Contains(g, token) {
			return false
		}
	}
	if strings.Contains(w, "microsoftedge") && strings.Contains(g, "edgewebview") {
		return false
	}
	return true
}

func registryNameScore(a, b string) int {
	if !registryNamesCompatible(a, b) {
		return 0
	}
	if normalizeSearch(a) == normalizeSearch(b) && normalizeSearch(a) != "" {
		return 100
	}
	ca, cb := canonicalRegistryName(a), canonicalRegistryName(b)
	if ca != "" && ca == cb {
		return 96
	}
	return 0
}

func exactSearchMatches(query string, rows []searchRow) ([]searchRow, bool) {
	want := normalizeSearch(query)
	if want == "" {
		return nil, false
	}
	seen := map[string]bool{}
	var out []searchRow
	for _, row := range rows {
		if normalizeSearch(row.Name) != want {
			continue
		}
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if !seen[key] {
			seen[key] = true
			out = append(out, row)
		}
	}
	return out, len(out) == 1
}
