package main

import "testing"

func TestExactSearchMatchesRejectsAmbiguity(t *testing.T) {
	rows := []searchRow{{Name: "Example App", ID: "Vendor.One"}, {Name: "Example App", ID: "Vendor.Two"}}
	matches, unique := exactSearchMatches("Example App", rows)
	if unique || len(matches) != 2 {
		t.Fatalf("ambiguous exact results accepted: unique=%v matches=%v", unique, matches)
	}
}

func TestExactSearchMatchesIgnoresFuzzySingleResult(t *testing.T) {
	rows := []searchRow{{Name: "Example App Helper", ID: "Vendor.Helper"}}
	matches, unique := exactSearchMatches("Example App", rows)
	if unique || len(matches) != 0 {
		t.Fatalf("fuzzy result accepted: unique=%v matches=%v", unique, matches)
	}
}

func TestRegistryNegativeMatching(t *testing.T) {
	cases := [][2]string{
		{"Microsoft Edge", "Microsoft Edge WebView2 Runtime"},
		{"1Password", "1Password Beta"},
		{"GitHub", "GitHub CLI"},
		{"Example", "Example Updater"},
		{"Example", "Example SDK"},
	}
	for _, c := range cases {
		if registryNameScore(c[0], c[1]) != 0 {
			t.Fatalf("unsafe registry match accepted: %q -> %q", c[0], c[1])
		}
	}
}

func TestRegistryArchitectureDecorationMatching(t *testing.T) {
	if score := registryNameScore("Mozilla Firefox", "Mozilla Firefox (x64 en-US)"); score < 96 {
		t.Fatalf("Firefox decorated name did not match safely: %d", score)
	}
	if score := registryNameScore("Zoom Workplace", "Zoom Workplace (64-bit)"); score < 96 {
		t.Fatalf("Zoom decorated name did not match safely: %d", score)
	}
}

func TestCandidateQueriesAreExplicit(t *testing.T) {
	q := candidateQueries("Zoom")
	if len(q) != 2 || q[0] != "Zoom" || q[1] != "Zoom Workplace" {
		t.Fatalf("unexpected Zoom aliases: %v", q)
	}
	q = candidateQueries("RealVNC Viewer")
	if len(q) != 1 || q[0] != "RealVNC Viewer" {
		t.Fatalf("generic destructive aliasing returned: %v", q)
	}
}
