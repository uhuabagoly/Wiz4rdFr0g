//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type managedPython struct {
	Key            string                            `json:"key"`
	Version        string                            `json:"version"`
	URL            string                            `json:"url"`
	Implementation string                            `json:"implementation"`
	Variant        string                            `json:"variant"`
	VersionParts   struct{ Major, Minor, Patch int } `json:"version_parts"`
}

func managedPythonDownloads(ctx context.Context) ([]managedPython, error) {
	out, err := exec.CommandContext(ctx, "uv", "--no-config", "python", "list", "--only-downloads", "--output-format", "json").Output()
	if err != nil {
		return nil, err
	}
	var rows []managedPython
	if err = json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func managedPythonSource(address string) string {
	for _, prefix := range []string{"https://github.com/astral-sh/python-build-standalone/releases/download/", "https://releases.astral.sh/github/python-build-standalone/releases/download/"} {
		if strings.HasPrefix(address, prefix) && len(address) > len(prefix) {
			return strings.TrimPrefix(address, prefix)
		}
	}
	return ""
}

func newestManagedPython(rows []managedPython) (managedPython, bool) {
	var selected managedPython
	for _, row := range rows {
		if row.Implementation != "cpython" || row.VersionParts.Major != 3 || !regexp.MustCompile(`^3\.[0-9]+\.[0-9]+$`).MatchString(row.Version) || (row.Variant != "" && row.Variant != "default") || managedPythonSource(row.URL) == "" {
			continue
		}
		if selected.Key == "" || row.VersionParts.Minor > selected.VersionParts.Minor || (row.VersionParts.Minor == selected.VersionParts.Minor && row.VersionParts.Patch > selected.VersionParts.Patch) {
			selected = row
		}
	}
	return selected, selected.Key != ""
}

func downloadManagedPython(ctx context.Context, key, directory string) (map[string]any, error) {
	proof := map[string]any{}
	rows, err := managedPythonDownloads(ctx)
	if err != nil {
		return proof, err
	}
	var selected managedPython
	for _, row := range rows {
		if row.Key == key {
			selected = row
			break
		}
	}
	if selected.URL == "" {
		return proof, fmt.Errorf("exact managed Python download missing")
	}
	versionOutput, err := exec.CommandContext(ctx, "uv", "--version").Output()
	if err != nil {
		return proof, err
	}
	version := regexp.MustCompile(`^uv ([0-9]+\.[0-9]+\.[0-9]+)`).FindStringSubmatch(string(versionOutput))
	if len(version) != 2 {
		return proof, fmt.Errorf("uv release version not resolved")
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	get := func(address string) (*http.Response, error) {
		req, e := http.NewRequestWithContext(ctx, "GET", address, nil)
		if e != nil {
			return nil, e
		}
		resp, e := client.Do(req)
		if e == nil && resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("Python source HTTP %d", resp.StatusCode)
		}
		return resp, e
	}
	metadataURL := "https://raw.githubusercontent.com/astral-sh/uv/" + version[1] + "/crates/uv-python/download-metadata.json"
	resp, err := get(metadataURL)
	if err != nil {
		return proof, err
	}
	var metadata map[string]struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 20<<20)).Decode(&metadata)
	resp.Body.Close()
	if err != nil {
		return proof, err
	}
	entry, ok := metadata[key]
	if !ok || managedPythonSource(selected.URL) == "" || managedPythonSource(entry.URL) != managedPythonSource(selected.URL) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(entry.SHA256) {
		return proof, fmt.Errorf("Python source and installed uv release checksum disagree")
	}
	proof["checksum_source"], proof["expected_sha256"], proof["resolved_download_url"], proof["expected_version"], proof["resolved_version"] = metadataURL, entry.SHA256, selected.URL, selected.Version, selected.Version
	resp, err = get(selected.URL)
	if err != nil {
		return proof, err
	}
	proof["download_http_status"], proof["final_download_url"], proof["content_type"] = resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type")
	path := filepath.Join(directory, "python.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		resp.Body.Close()
		return proof, err
	}
	digest := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, digest), io.LimitReader(resp.Body, 512<<20))
	file.Close()
	resp.Body.Close()
	actual := hex.EncodeToString(digest.Sum(nil))
	proof["downloaded_bytes"], proof["sha256"] = size, actual
	if err != nil || size == 0 || actual != entry.SHA256 {
		return proof, fmt.Errorf("Python archive size/checksum invalid: %v", err)
	}
	out, err := exec.CommandContext(ctx, "tar", "-tzf", path).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "python/bin/python3") {
		return proof, fmt.Errorf("Python archive lacks actual interpreter payload")
	}
	proof["file_validation"], proof["download_real"] = true, true
	return proof, nil
}
