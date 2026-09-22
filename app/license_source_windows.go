//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

// This read-only source collection is NOT physical lifecycle evidence. The
// catalog policy remains unchanged until the retrieved sources are reviewed.
func runLicenseSourceReview(indexFile, directory string) int {
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return 2
	}
	var indexes []int
	if json.Unmarshal(data, &indexes) != nil {
		return 2
	}
	if err = os.MkdirAll(directory, 0755); err != nil {
		return 2
	}
	for _, index := range indexes {
		if index < 0 || index >= len(catalog) {
			return 2
		}
		app := catalog[index]
		record := map[string]any{"kind": "license source review; not physical evidence", "catalog_index": index, "app_name": app.Name, "git_commit": os.Getenv("GITHUB_SHA"), "test_run_id": os.Getenv("GITHUB_RUN_ID"), "reviewed_at": time.Now().UTC().Format(time.RFC3339), "review_status": "UNRESOLVED"}
		profile := catalogpkg.ProfileFor(app)
		if profile.License.Class != catalogpkg.LicenseUnknown {
			record["review_status"] = "EXISTING_CLASSIFICATION"
			record["metadata"] = profile.License
		} else {
			collectLicenseSource(app, index, directory, record)
		}
		output, _ := json.MarshalIndent(record, "", "  ")
		if os.WriteFile(filepath.Join(directory, fmt.Sprintf("%04d.json", index)), output, 0644) != nil {
			return 2
		}
	}
	return 0
}

func collectLicenseSource(app appDef, index int, directory string, record map[string]any) {
	id, source, _, err := loadPackageData(app)
	record["package_id"] = id
	record["package_source"] = source
	if err != nil || id == "" || source != "winget" {
		record["failure"] = fmt.Sprintf("exact production resolution: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	metadata, err := runWinget(ctx, "show", "--id", id, "--exact", "--source", source, "--accept-source-agreements", "--disable-interactivity")
	record["package_metadata"] = metadata
	if err != nil {
		record["failure"] = err.Error()
		return
	}
	field := func(key string) string {
		m := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `:\s*([^\r\n]+)`).FindStringSubmatch(metadata)
		if len(m) == 2 {
			return strings.TrimSpace(m[1])
		}
		return ""
	}
	licenseURL := field("License Url")
	record["license_label"] = field("License")
	record["publisher_url"] = field("Publisher Url")
	record["license_url"] = licenseURL
	uri, err := url.Parse(licenseURL)
	if err != nil || uri.Scheme != "https" || uri.Host == "" || uri.User != nil {
		record["failure"] = "no exact HTTPS license URL"
		return
	}
	if uri.Host == "github.com" {
		parts := strings.Split(strings.Trim(uri.Path, "/"), "/")
		if len(parts) >= 5 && parts[2] == "blob" {
			uri.Host = "raw.githubusercontent.com"
			uri.Path = "/" + strings.Join(append(parts[:2], parts[3:]...), "/")
			uri.RawQuery = ""
		}
	}
	record["retrieved_url"] = uri.String()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		record["failure"] = err.Error()
		return
	}
	response, err := (&http.Client{Timeout: 45 * time.Second}).Do(request)
	if err != nil {
		record["failure"] = err.Error()
		return
	}
	defer response.Body.Close()
	record["http_status"] = response.StatusCode
	record["final_url"] = response.Request.URL.String()
	record["content_type"] = response.Header.Get("Content-Type")
	if response.StatusCode != 200 {
		record["failure"] = "license source HTTP failure"
		return
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(body) > 2*1024*1024 || len(body) == 0 {
		record["failure"] = "license source unreadable, empty or too large"
		return
	}
	digest := sha256.Sum256(body)
	record["source_sha256"] = hex.EncodeToString(digest[:])
	record["source_bytes"] = len(body)
	path := fmt.Sprintf("%04d-license.txt", index)
	if os.WriteFile(filepath.Join(directory, path), body, 0644) != nil {
		record["failure"] = "license source archive could not be saved"
		return
	}
	record["source_file"] = path
	record["review_status"] = "SOURCE_RETRIEVED"
	text := strings.Join(strings.Fields(string(body)), " ")
	if strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
		return
	}
	label := strings.ToLower(field("License"))
	matched := ""
	switch {
	case strings.Contains(label, "mit") && strings.Contains(text, "Permission is hereby granted, free of charge, to any person obtaining a copy"):
		matched = "MIT"
	case strings.Contains(label, "gpl") && strings.Contains(text, "GNU GENERAL PUBLIC LICENSE"):
		matched = "GPL"
	case strings.Contains(label, "apache") && strings.Contains(text, "Apache License") && strings.Contains(text, "Grant of Copyright License"):
		matched = "Apache-2.0"
	case strings.Contains(label, "mpl") && strings.Contains(text, "Mozilla Public License") && strings.Contains(text, "Each Contributor hereby grants"):
		matched = "MPL-2.0"
	case strings.Contains(label, "bsd") && strings.Contains(text, "Redistribution and use in source and binary forms") && strings.Contains(text, "with or without modification"):
		matched = "BSD"
	}
	if matched != "" {
		record["standard_license_text_match"] = matched
	}
}
