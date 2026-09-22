//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const veraCryptFingerprint = "5069A233D55A0EEB174A5FC3821ACD02680D16DE"

func veraCryptPlatform() (string, error) {
	body, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			values[key] = strings.Trim(value, `"`)
		}
	}
	distro := map[string]string{"ubuntu": "Ubuntu", "debian": "Debian"}[values["ID"]]
	arch := map[string]string{"amd64": "amd64", "arm64": "arm64", "386": "i386"}[runtime.GOARCH]
	if distro == "" || arch == "" || !regexp.MustCompile(`^[0-9.]+$`).MatchString(values["VERSION_ID"]) {
		return "", fmt.Errorf("no official VeraCrypt DEB platform match")
	}
	return distro + "-" + values["VERSION_ID"] + "-" + arch, nil
}

func veraCryptLinks(page, platform string) (string, string, error) {
	pattern := regexp.MustCompile(`^veracrypt-([0-9.]+)-` + regexp.QuoteMeta(platform) + `\.deb$`)
	links := map[string]bool{}
	for _, match := range regexp.MustCompile(`(?i)href=["']([^"']+)["']`).FindAllStringSubmatch(page, -1) {
		links[html.UnescapeString(match[1])] = true
	}
	var chosen, version string
	for link := range links {
		u, err := url.Parse(link)
		if err != nil || u.Scheme != "https" || u.Host != "github.com" || !strings.HasPrefix(strings.ToLower(u.Path), "/veracrypt/veracrypt/releases/download/") {
			continue
		}
		m := pattern.FindStringSubmatch(filepath.Base(u.Path))
		if len(m) != 2 {
			continue
		}
		if chosen != "" {
			return "", "", fmt.Errorf("ambiguous official VeraCrypt release")
		}
		chosen, version = link, m[1]
	}
	if chosen == "" || !links[chosen+".sig"] {
		return "", "", fmt.Errorf("official platform DEB and detached signature not published")
	}
	return chosen, version, nil
}

// Download the genuine publisher DEB, authenticate its detached signature with
// the pinned publisher key, and inspect dpkg identity before apt may install it.
func prepareVeraCryptDeb(ctx context.Context, directory string) (string, map[string]any, error) {
	proof := map[string]any{}
	client := &http.Client{Timeout: 15 * time.Minute}
	get := func(address string, limit int64) ([]byte, *http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", address, nil)
		if err != nil {
			return nil, nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, resp, fmt.Errorf("vendor download HTTP %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if err != nil {
			return nil, resp, err
		}
		if len(body) == 0 || int64(len(body)) > limit {
			return nil, resp, fmt.Errorf("vendor file empty or oversized")
		}
		return body, resp, nil
	}
	platform, err := veraCryptPlatform()
	if err != nil {
		return "", proof, err
	}
	page, _, err := get("https://veracrypt.io/en/Downloads.html", 2<<20)
	if err != nil {
		return "", proof, err
	}
	address, version, err := veraCryptLinks(string(page), platform)
	if err != nil {
		return "", proof, err
	}
	proof["resolved_download_url"], proof["expected_version"] = address, version
	body, resp, err := get(address, 256<<20)
	if resp != nil {
		proof["download_http_status"] = resp.StatusCode
		proof["final_download_url"] = resp.Request.URL.String()
		proof["content_type"] = resp.Header.Get("Content-Type")
	}
	if err != nil {
		return "", proof, err
	}
	digest := sha256.Sum256(body)
	proof["downloaded_bytes"], proof["sha256"] = len(body), hex.EncodeToString(digest[:])
	if !strings.HasPrefix(string(body[:min(len(body), 8)]), "!<arch>\n") {
		return "", proof, fmt.Errorf("vendor download is not a DEB archive")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", proof, err
	}
	path := filepath.Join(directory, "veracrypt.deb")
	if err := os.WriteFile(path, body, 0600); err != nil {
		return "", proof, err
	}
	sig, _, err := get(address+".sig", 1<<20)
	if err != nil {
		return "", proof, err
	}
	key, _, err := get("https://amcrypto.jp/VeraCrypt/VeraCrypt_PGP_public_key.asc", 1<<20)
	if err != nil {
		return "", proof, err
	}
	for file, data := range map[string][]byte{"publisher.asc": key, "veracrypt.deb.sig": sig} {
		if err := os.WriteFile(filepath.Join(directory, file), data, 0600); err != nil {
			return "", proof, err
		}
	}
	keyHome := filepath.Join(directory, "keyring")
	if err := os.MkdirAll(keyHome, 0700); err != nil {
		return "", proof, err
	}
	gpg := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "gpg", append([]string{"--batch", "--homedir", keyHome}, args...)...).CombinedOutput()
	}
	out, err := gpg("--import", filepath.Join(directory, "publisher.asc"))
	if err != nil {
		return "", proof, fmt.Errorf("publisher key import: %w: %s", err, out)
	}
	out, err = gpg("--status-fd", "1", "--verify", path+".sig", path)
	proof["signature_verification"] = string(out)
	if err != nil {
		return "", proof, fmt.Errorf("VeraCrypt detached signature invalid: %w", err)
	}
	valid := false
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 3 && fields[0] == "[GNUPG:]" && fields[1] == "VALIDSIG" && (fields[2] == veraCryptFingerprint || fields[len(fields)-1] == veraCryptFingerprint) {
			valid = true
		}
	}
	if !valid {
		return "", proof, fmt.Errorf("VeraCrypt signature does not match pinned publisher fingerprint")
	}
	out, err = exec.CommandContext(ctx, "dpkg-deb", "-f", path, "Package", "Version").CombinedOutput()
	proof["package_validation"] = string(out)
	if err != nil || !strings.Contains(string(out), "Package: veracrypt\n") {
		return "", proof, fmt.Errorf("vendor DEB package identity invalid: %s", out)
	}
	resolved := ""
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Version: ") {
			resolved = strings.TrimPrefix(line, "Version: ")
		}
	}
	if resolved != version && !strings.HasPrefix(resolved, version+"-") {
		return "", proof, fmt.Errorf("vendor DEB release version mismatch: %s", resolved)
	}
	proof["resolved_version"], proof["signature_verified"], proof["file_validation"], proof["download_real"] = resolved, true, true, true
	return path, proof, nil
}
