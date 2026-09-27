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
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
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

func vmVerifyHTTPDownload(ctx context.Context, id, source, version, workRoot, scope string) (vmDownloadProof, error) {
	var proof vmDownloadProof
	args := []string{"show", "--id", id, "--exact", "--source", source, "--version", version, "--accept-source-agreements", "--disable-interactivity"}
	if scope != "" {
		args = append(args, "--scope", scope)
	}
	code, out, err := runDirectProcess(ctx, "winget.exe", args)
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
	if response.StatusCode == 200 && response.Request.URL.Host == "sourceforge.net" && strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
		page, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		redirect := sourceForgeDownloadRedirect(proof.URL, page)
		if readErr != nil || redirect == "" {
			return proof, fmt.Errorf("SourceForge download page has no verified exact-file redirect")
		}
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, redirect, nil)
		if err != nil {
			return proof, err
		}
		response, err = (&http.Client{Timeout: 20 * time.Minute}).Do(request)
		if err != nil {
			return proof, err
		}
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
	RemovalDiagnostics   map[string]string       `json:"removal_diagnostics,omitempty"`
	RelatedRegistrations []registryPackage       `json:"related_registrations,omitempty"`
	Appx                 *releaseproof.AppxProof `json:"appx,omitempty"`
	Registration         registryPackage         `json:"registration"`
	RegistryKey          string                  `json:"registry_key"`
	BinaryPaths          []string                `json:"binary_paths"`
	UpdaterPaths         []string                `json:"updater_paths,omitempty"`
	RegistryPresent      bool                    `json:"registry_present"`
	BinariesPresent      bool                    `json:"binaries_present"`
	RegistryRemoved      bool                    `json:"registry_removed"`
	BinariesRemoved      bool                    `json:"binaries_removed"`
	PayloadRegistry      string                  `json:"payload_registry,omitempty"`
	PayloadVersion       string                  `json:"payload_version,omitempty"`
}

// Record observations even when the production uninstaller fails before the
// independent removal gate. These diagnostics never set any PASS flags.
func vmCaptureRemovalDiagnostics(proof *vmFilesystemProof) {
	proof.RemovalDiagnostics = make(map[string]string)
	for _, path := range proof.BinaryPaths {
		info, err := os.Stat(path)
		switch {
		case os.IsNotExist(err):
			proof.RemovalDiagnostics[path] = "absent"
		case err != nil:
			proof.RemovalDiagnostics[path] = "unknown: " + err.Error()
		default:
			proof.RemovalDiagnostics[path] = fmt.Sprintf("present: size=%d regular=%t", info.Size(), info.Mode().IsRegular())
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	regs := append([]registryPackage{proof.Registration}, proof.RelatedRegistrations...)
	for _, reg := range regs {
		if reg.RegistryKey == "" {
			continue
		}
		args := []string{"query", reg.RegistryKey}
		if reg.RegistryView != "" {
			args = append(args, reg.RegistryView)
		}
		code, out, err := runDirectProcess(ctx, "reg.exe", args)
		proof.RemovalDiagnostics[reg.RegistryKey+" "+reg.RegistryView] = fmt.Sprintf("exit=%d error=%v output=%s", code, err, strings.TrimSpace(out))
	}
}

func vmCaptureFilesystem(ctx context.Context, detected vmDetected, catalogName, requestedScope string) (vmFilesystemProof, error) {
	var proof vmFilesystemProof
	name := detected.Name
	registrations := scanRegistryPackages()
	reg, state := bestRegistryMatchDetailed(name, registrations)
	if state == registryMatchNone {
		reg, state = registryForTruncatedPackage(installedPackage{Name: name, Version: detected.Version, Scope: detected.Scope}, registrations)
	}
	if state == registryMatchNone {
		for _, alias := range candidateQueries(catalogName) {
			reg, state = bestRegistryMatchDetailed(alias, registrations)
			if state != registryMatchNone {
				break
			}
		}
	}
	if state != registryMatchFound {
		return vmCaptureAppx(ctx, name)
	}
	proof.RegistryKey = reg.RegistryKey
	proof.Registration = reg
	for _, related := range registrations {
		if sameRegisteredProduct(reg, related) {
			proof.RelatedRegistrations = append(proof.RelatedRegistrations, related)
		}
	}
	args := []string{"query", reg.RegistryKey}
	if reg.RegistryView != "" {
		args = append(args, reg.RegistryView)
	}
	code, output, err := runDirectProcess(ctx, "reg.exe", args)
	if err != nil || code != 0 {
		return proof, fmt.Errorf("independent registry presence query failed (exit %d, view %s): %s", code, reg.RegistryView, compactLog(output))
	}
	proof.RegistryPresent = true
	// Burn's DisplayIcon may be the cached installer itself. Python's PEP 514
	// registration identifies the interpreter independently of that cache.
	if pythonVersion := regexp.MustCompile(`^Python (\d+\.\d+\.\d+) \(64-bit\)$`).FindStringSubmatch(reg.DisplayName); len(pythonVersion) == 2 {
		version := pythonVersion[1]
		parts := strings.Split(version, ".")
		hive := strings.SplitN(reg.RegistryKey, `\`, 2)[0]
		// The Burn bundle can register in HKCU while its explicitly requested
		// all-users MSI payload is registered in HKLM. Keep the bundle proof
		// intact and inspect PEP 514 in the actual requested payload scope.
		if requestedScope == "machine" {
			hive = "HKEY_LOCAL_MACHINE"
		} else if requestedScope == "user" {
			hive = "HKEY_CURRENT_USER"
		}
		key := hive + `\Software\Python\PythonCore\` + parts[0] + "." + parts[1] + `\InstallPath`
		query := []string{"query", key, "/v", "ExecutablePath"}
		if reg.RegistryView != "" {
			query = append(query, reg.RegistryView)
		}
		code, output, err := runDirectProcess(ctx, "reg.exe", query)
		match := regexp.MustCompile(`(?m)^\s*ExecutablePath\s+REG_SZ\s+(.+?)\s*$`).FindStringSubmatch(output)
		if err != nil || code != 0 || len(match) != 2 {
			return proof, fmt.Errorf("Python interpreter registration unavailable: %s", compactLog(output))
		}
		path := strings.TrimSpace(match[1])
		if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Base(path), "python.exe") || !vmApplicationPayloadPath(path) {
			return proof, fmt.Errorf("Python registration does not identify an interpreter: %s", path)
		}
		code, output, err = runDirectProcess(ctx, path, []string{"-I", "--version"})
		if err != nil || code != 0 || strings.TrimSpace(output) != "Python "+version {
			return proof, fmt.Errorf("registered Python interpreter version mismatch: %s", compactLog(output))
		}
		proof.PayloadRegistry, proof.PayloadVersion = key, strings.TrimSpace(output)
		proof.BinaryPaths = append(proof.BinaryPaths, path)
	}
	icon := executablePathFromRegistryValue(reg.DisplayIcon)
	if len(proof.BinaryPaths) == 0 && strings.EqualFold(filepath.Ext(icon), ".exe") && vmApplicationPayloadPath(icon) {
		if info, err := os.Stat(icon); err == nil && !info.IsDir() {
			proof.BinaryPaths = append(proof.BinaryPaths, icon)
		}
	}
	if len(proof.BinaryPaths) == 0 {
		if reg.WindowsInstaller != 0 {
			paths, err := vmMSIExecutableComponents(ctx, filepath.Base(reg.RegistryKey))
			if err != nil {
				return proof, err
			}
			proof.BinaryPaths = append(proof.BinaryPaths, paths...)
		}
	}
	if len(proof.BinaryPaths) == 0 {
		root := deriveRegistryInstallLocation(reg)
		if filepath.IsAbs(root) && vmApplicationPayloadPath(root) && filepath.Clean(root) != filepath.VolumeName(root)+string(filepath.Separator) {
			err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				lower := strings.ToLower(entry.Name())
				// Squirrel.exe is the bundled updater, not the application payload.
				// Require its registered Squirrel uninstall mechanism before treating
				// it as auxiliary; preserve the observed path in the evidence.
				if entry.Type().IsRegular() && lower == "squirrel.exe" && strings.Contains(strings.ToLower(reg.UninstallString), "update.exe") && strings.Contains(strings.ToLower(reg.UninstallString), "--uninstall") {
					proof.UpdaterPaths = append(proof.UpdaterPaths, path)
					return nil
				}
				if entry.Type().IsRegular() && strings.HasSuffix(lower, ".exe") && vmApplicationPayloadPath(path) {
					proof.BinaryPaths = append(proof.BinaryPaths, path)
				}
				return nil
			})
			if err != nil {
				return proof, fmt.Errorf("inspect registered installation directory: %w", err)
			}
		}
	}
	proof.BinariesPresent = len(proof.BinaryPaths) > 0
	if !proof.BinariesPresent {
		return proof, fmt.Errorf("no independently observed application executable")
	}
	return proof, nil
}

// Query MSI's installed component key paths for this exact ProductCode. This
// does not invoke Win32_Product or MSI repair, and never guesses an install path.
func vmMSIExecutableComponents(ctx context.Context, productCode string) ([]string, error) {
	if !regexp.MustCompile(`^\{[0-9A-Fa-f-]{36}\}$`).MatchString(productCode) {
		return nil, fmt.Errorf("invalid MSI product code")
	}
	product, err := syscall.UTF16PtrFromString(productCode)
	if err != nil {
		return nil, err
	}
	msi := syscall.NewLazyDLL("msi.dll")
	enumerate := msi.NewProc("MsiEnumComponentsW")
	componentPath := msi.NewProc("MsiGetComponentPathW")
	if err = enumerate.Find(); err != nil {
		return nil, err
	}
	if err = componentPath.Find(); err != nil {
		return nil, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var component [39]uint16
	var path [32768]uint16
	var paths []string
	seen := map[string]bool{}
	for index := uint32(0); ; index++ {
		if err = ctx.Err(); err != nil {
			return paths, err
		}
		code, _, _ := enumerate.Call(uintptr(index), uintptr(unsafe.Pointer(&component[0])))
		if code == 259 {
			break
		}
		if code != 0 {
			return paths, fmt.Errorf("MSI component enumeration returned %d", code)
		}
		length := uint32(len(path))
		state, _, _ := componentPath.Call(uintptr(unsafe.Pointer(product)), uintptr(unsafe.Pointer(&component[0])), uintptr(unsafe.Pointer(&path[0])), uintptr(unsafe.Pointer(&length)))
		if state != 3 || length == 0 || length >= uint32(len(path)) {
			continue
		}
		value := syscall.UTF16ToString(path[:length])
		if !strings.EqualFold(filepath.Ext(value), ".exe") || !vmApplicationPayloadPath(value) || seen[strings.ToLower(value)] {
			continue
		}
		if info, err := os.Stat(value); err == nil && info.Mode().IsRegular() {
			paths = append(paths, value)
			seen[strings.ToLower(value)] = true
		}
	}
	return paths, nil
}

func vmVerifyFilesystemRemoved(ctx context.Context, proof *vmFilesystemProof) error {
	if proof.Appx != nil {
		items, err := vmReadAppxFamily(ctx, proof.Appx.Family)
		if err != nil {
			return err
		}
		if len(items) != 0 {
			return fmt.Errorf("Appx deployment registration remains")
		}
		proof.Appx.Removed = true
	} else {
		args := []string{"query", proof.RegistryKey}
		if proof.Registration.RegistryView != "" {
			args = append(args, proof.Registration.RegistryView)
		}
		code, out, err := runDirectProcess(ctx, "reg.exe", args)
		if err == nil || code != 1 || !strings.Contains(strings.ToLower(out), "unable to find") {
			return fmt.Errorf("registry disappearance is not proven")
		}
		proof.RegistryRemoved = true
		for _, related := range proof.RelatedRegistrations {
			args := []string{"query", related.RegistryKey}
			if related.RegistryView != "" {
				args = append(args, related.RegistryView)
			}
			code, out, err := runDirectProcess(ctx, "reg.exe", args)
			if err == nil || code != 1 || !strings.Contains(strings.ToLower(out), "unable to find") {
				proof.RegistryRemoved = false
				return fmt.Errorf("related wrapper registration remains: %s", related.RegistryKey)
			}
		}
	}
	// Self-relocating uninstallers can remove registration before their child
	// finishes deleting payloads. Observe bounded completion; never delete here.
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := awaitApplicationFilesRemoved(waitCtx, proof.BinaryPaths, time.Second); err != nil {
		return err
	}
	proof.BinariesRemoved = true
	return nil
}
