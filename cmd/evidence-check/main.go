package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
	"wiz4rdfr0g.local/fullcatalog/internal/releasegate"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: evidence-check <result.json> <build_manifest.json>")
		os.Exit(2)
	}
	resultPath, manifestPath := os.Args[1], os.Args[2]
	manifest, err := releasegate.LoadManifest(manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(3)
	}
	b, err := os.ReadFile(resultPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "result:", err)
		os.Exit(4)
	}
	var r releasegate.Result
	if err := json.Unmarshal(b, &r); err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(5)
	}
	entries := catalogpkg.BuildAuditEntries()
	if r.CatalogIndex < 0 || r.CatalogIndex >= len(entries) {
		fmt.Fprintln(os.Stderr, "catalog index out of range")
		os.Exit(6)
	}
	fp, err := releaseproof.CatalogFingerprint(entries)
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog fingerprint:", err)
		os.Exit(7)
	}
	if issues := releaseproof.ValidateManifest(manifest, fp); len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "manifest %s: %s\n", issue.Code, issue.Message)
		}
		os.Exit(8)
	}
	key := []byte(os.Getenv(releaseproof.EvidenceKeyEnvironment))
	issues := releaseproof.ValidateEvidence(r.EvidenceStatement, r.Signature, manifest, entries[r.CatalogIndex], time.Now().UTC(), key)
	if len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "%s: %s\n", issue.Code, issue.Message)
		}
		os.Exit(9)
	}
	if strings.TrimSpace(r.FinalStatus) == "" {
		fmt.Fprintln(os.Stderr, "empty final status")
		os.Exit(10)
	}
	fmt.Printf("VALID index=%d status=%s build_id=%s\n", r.CatalogIndex, r.FinalStatus, r.BuildID)
}
