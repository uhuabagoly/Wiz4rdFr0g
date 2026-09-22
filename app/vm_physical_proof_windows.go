//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type vmDownloadProof struct {
	URL            string `json:"resolved_download_url"`
	FinalURL       string `json:"final_download_url"`
	HTTPStatus     int    `json:"download_http_status"`
	Bytes          int64  `json:"downloaded_bytes"`
	SHA256         string `json:"sha256"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ContentType    string `json:"content_type"`
	Format         string `json:"file_format"`
	Valid          bool   `json:"file_validation"`
}

func vmVerifyHTTPDownload(ctx context.Context, id, source, version, workRoot string) (vmDownloadProof, error) {
	var proof vmDownloadProof
	code, out, err := runDirectProcess(ctx, "winget.exe", []string{"show", "--id", id, "--exact", "--source", source, "--version", version, "--accept-source-agreements", "--disable-interactivity"})
	if err != nil || code != 0 {
		return proof, fmt.Errorf("exact installer metadata unavailable: exit=%d %v", code, err)
	}
	urls := regexp.MustCompile(`(?m)^\s*Installer Url:\s*(https?://\S+)`).FindStringSubmatch(out)
	sums := regexp.MustCompile(`(?m)^\s*Installer SHA256:\s*([a-fA-F0-9]{64})`).FindStringSubmatch(out)
	if len(urls) != 2 || len(sums) != 2 {
		return proof, fmt.Errorf("missing exact installer URL/SHA256")
	}
	proof.URL = urls[1]
	proof.ExpectedSHA256 = strings.ToLower(sums[1])
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, proof.URL, nil)
	if err != nil {
		return proof, err
	}
	response, err := (&http.Client{Timeout: 20 * time.Minute}).Do(request)
	if err != nil {
		return proof, err
	}
	defer response.Body.Close()
	proof.HTTPStatus = response.StatusCode
	proof.FinalURL = response.Request.URL.String()
	proof.ContentType = response.Header.Get("Content-Type")
	if response.StatusCode != 200 {
		return proof, fmt.Errorf("installer HTTP %d", response.StatusCode)
	}
	f, err := os.OpenFile(filepath.Join(workRoot, "http-verified-installer.bin"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		return proof, err
	}
	defer f.Close()
	hash := sha256.New()
	proof.Bytes, err = io.Copy(io.MultiWriter(f, hash), response.Body)
	proof.SHA256 = hex.EncodeToString(hash.Sum(nil))
	if err != nil {
		return proof, err
	}
	if proof.Bytes == 0 || proof.SHA256 != proof.ExpectedSHA256 {
		return proof, fmt.Errorf("installer size/SHA256 mismatch")
	}
	if _, err = f.Seek(0, 0); err != nil {
		return proof, err
	}
	var header [8]byte
	if _, err = io.ReadFull(f, header[:]); err != nil {
		return proof, err
	}
	switch {
	case string(header[:2]) == "MZ":
		proof.Format = "PE"
	case string(header[:4]) == "PK\x03\x04":
		proof.Format = "ZIP/MSIX"
	case string(header[:8]) == "\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1":
		proof.Format = "MSI/OLE"
	default:
		return proof, fmt.Errorf("download is not a supported binary installer format")
	}
	proof.Valid = true
	return proof, nil
}

type vmFilesystemProof struct {
	RegistryKey     string   `json:"registry_key"`
	BinaryPaths     []string `json:"binary_paths"`
	RegistryPresent bool     `json:"registry_present"`
	BinariesPresent bool     `json:"binaries_present"`
	RegistryRemoved bool     `json:"registry_removed"`
	BinariesRemoved bool     `json:"binaries_removed"`
}

func vmCaptureFilesystem(ctx context.Context, name string) (vmFilesystemProof, error) {
	var proof vmFilesystemProof
	reg, ok := bestRegistryMatch(name, scanRegistryPackages())
	if !ok {
		return proof, fmt.Errorf("no unambiguous independent uninstall registration")
	}
	proof.RegistryKey = reg.RegistryKey
	code, _, err := runDirectProcess(ctx, "reg.exe", []string{"query", reg.RegistryKey})
	if err != nil || code != 0 {
		return proof, fmt.Errorf("independent registry presence query failed")
	}
	proof.RegistryPresent = true
	icon := executablePathFromRegistryValue(reg.DisplayIcon)
	if strings.EqualFold(filepath.Ext(icon), ".exe") {
		if info, err := os.Stat(icon); err == nil && !info.IsDir() {
			proof.BinaryPaths = append(proof.BinaryPaths, icon)
		}
	}
	if len(proof.BinaryPaths) == 0 {
		root := deriveRegistryInstallLocation(reg)
		if filepath.IsAbs(root) && filepath.Clean(root) != filepath.VolumeName(root)+string(filepath.Separator) {
			entries, _ := os.ReadDir(root)
			for _, entry := range entries {
				lower := strings.ToLower(entry.Name())
				if !entry.IsDir() && strings.HasSuffix(lower, ".exe") && !strings.Contains(lower, "unins") && !strings.Contains(lower, "update") {
					proof.BinaryPaths = append(proof.BinaryPaths, filepath.Join(root, entry.Name()))
				}
			}
		}
	}
	proof.BinariesPresent = len(proof.BinaryPaths) > 0
	if !proof.BinariesPresent {
		return proof, fmt.Errorf("no independently observed application executable")
	}
	return proof, nil
}

func vmVerifyFilesystemRemoved(ctx context.Context, proof *vmFilesystemProof) error {
	code, out, err := runDirectProcess(ctx, "reg.exe", []string{"query", proof.RegistryKey})
	if err == nil || code != 1 || !strings.Contains(strings.ToLower(out), "unable to find") {
		return fmt.Errorf("registry disappearance is not proven")
	}
	proof.RegistryRemoved = true
	for _, path := range proof.BinaryPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return fmt.Errorf("application executable still exists or cannot be checked: %s", path)
		}
	}
	proof.BinariesRemoved = true
	return nil
}
