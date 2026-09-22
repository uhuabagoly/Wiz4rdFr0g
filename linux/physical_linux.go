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
	"runtime"
	"strconv"
	"strings"
	"time"

	"wiz4rdfr0g.local/fullcatalog/internal/linuxpkg"
)

type physicalCommand struct {
	Phase    string   `json:"phase"`
	Command  []string `json:"command"`
	ExitCode int      `json:"exit_code"`
	Output   string   `json:"output"`
	Started  string   `json:"started_at"`
	Finished string   `json:"finished_at"`
}

func linuxLifecycle(index int, resultPath string) int {
	r := map[string]any{"test_run_id": os.Getenv("GITHUB_RUN_ID"), "attempt": os.Getenv("GITHUB_RUN_ATTEMPT"), "platform": "linux", "architecture": runtime.GOARCH, "git_commit": os.Getenv("GITHUB_SHA"), "catalog_index": index, "started_at": time.Now().UTC().Format(time.RFC3339Nano), "final_status": "FAIL", "failure_reason": "lifecycle did not complete", "download_real": false, "install_real": false, "independent_detection": false, "wiz4rd_detection": false, "uninstall_real": false, "independent_removed_detection": false, "wiz4rd_removed_detection": false, "download_http_status": nil, "expected_version": "", "resolved_version": "", "resolved_download_url": "", "downloaded_bytes": 0, "sha256": "", "file_validation": false, "install_exit_code": nil, "uninstall_exit_code": nil}
	r["campaign_id"] = os.Getenv("WIZ4RDFR0G_CAMPAIGN_ID")
	r["campaign_attempt"] = os.Getenv("WIZ4RDFR0G_CAMPAIGN_ATTEMPT")
	var commands []physicalCommand
	save := func() {
		r["commands"] = commands
		b, _ := json.MarshalIndent(r, "", "  ")
		if err := os.MkdirAll(filepath.Dir(resultPath), 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		f, err := os.CreateTemp(filepath.Dir(resultPath), ".physical-*")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		name := f.Name()
		if _, err = f.Write(b); err == nil {
			err = f.Sync()
		}
		f.Close()
		if err == nil {
			err = os.Rename(name, resultPath)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Remove(name)
		}
	}
	defer func() { r["finished_at"] = time.Now().UTC().Format(time.RFC3339Nano); save() }()
	fail := func(err error) int { r["failure_reason"] = err.Error(); fmt.Fprintln(os.Stderr, err); return 1 }
	if index < 0 || index >= len(candidates) {
		return fail(fmt.Errorf("invalid catalog index"))
	}
	candidate := candidates[index]
	identity := sha256.Sum256([]byte("linux\x00" + candidate.Name))
	r["app_id"] = hex.EncodeToString(identity[:])
	r["app_name"] = candidate.Name
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Getenv("RUNNER_OS") != "Linux" || os.Getenv("GITHUB_RUN_ID") == "" || os.Geteuid() == 0 {
		return fail(fmt.Errorf("fresh GitHub-hosted Linux user session required; never run on a personal machine"))
	}
	osRelease, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fail(err)
	}
	r["os_version"] = string(osRelease)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Minute)
	defer cancel()
	run := func(phase, name string, args ...string) (string, int, error) {
		r["phase"] = phase
		save()
		start := time.Now()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = append(os.Environ(), "LC_ALL=C", "DEBIAN_FRONTEND=noninteractive")
		out, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			code = -1
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			}
		}
		s := string(out)
		commands = append(commands, physicalCommand{phase, append([]string{name}, args...), code, s, start.UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano)})
		save()
		return s, code, err
	}
	distroName, packageManager = detectLinux()
	packageSet = loadPackageSet(packageManager)
	flatpakSet = loadFlatpakSet()
	var target *linuxApp
	for _, a := range resolveLinuxApps() {
		if a.Name == candidate.Name {
			copy := a
			target = &copy
			break
		}
	}
	if target == nil {
		return fail(fmt.Errorf("no explicit catalog package available on this runner for %s", candidate.Name))
	}
	a := *target
	r["provider"] = a.Provider
	r["resolved_package_id"] = a.Package
	r["uninstall_mechanism"] = a.Provider
	before, err := isInstalledExact(ctx, a)
	r["pre_install_detection"] = before
	if err != nil {
		return fail(err)
	}
	if before {
		// This entry point is restricted above to a disposable GitHub-hosted
		// runner. Remove only this exact catalog package via the production
		// package manager, preserving the preparation commands as evidence.
		r["runner_baseline_installed"] = true
		cleanup, err := operationSpec(a, "Legújabb", true)
		if err != nil {
			return fail(err)
		}
		if a.Provider == "apt-get" && a.Package == "git" {
			// The hosted image's Git PPA docs may be newer than the configured
			// repository. Remove the baseline's exact companion package too.
			cleanup.Args = append(cleanup.Args, "git-man")
		}
		if cleanup.NeedsRoot {
			_, _, err = run("PREPARE_CLEAN_RUNNER", "sudo", append([]string{"-n", "--", cleanup.Name}, cleanup.Args...)...)
		} else {
			_, _, err = run("PREPARE_CLEAN_RUNNER", cleanup.Name, cleanup.Args...)
		}
		if err != nil {
			return fail(fmt.Errorf("runner baseline cleanup: %w", err))
		}
		before, err = isInstalledExact(ctx, a)
		if err != nil || before {
			return fail(fmt.Errorf("runner baseline removal did not establish an absent state: %v", err))
		}
		r["pre_install_detection"] = false
	}
	work, err := os.MkdirTemp("", "Wiz4rdFr0g-linux-physical-")
	if err != nil {
		return fail(err)
	}
	defer os.RemoveAll(work)
	version := "Legújabb"
	if a.Provider == "ollama-vendor" {
		path, proof, err := downloadOllama(ctx, work, version)
		for key, value := range proof {
			r[key] = value
		}
		save()
		if err != nil {
			return fail(err)
		}
		a.PackageFile = path
		version = proof["resolved_version"].(string)
	} else if a.Provider == "uv-python" {
		proof, err := downloadManagedPython(ctx, a.Package, work)
		for key, value := range proof {
			r[key] = value
		}
		save()
		if err != nil {
			return fail(err)
		}
		version = proof["resolved_version"].(string)
	} else if a.VendorDEB {
		path, proof, err := prepareVeraCryptDeb(ctx, work)
		for key, value := range proof {
			r[key] = value
		}
		save()
		if err != nil {
			return fail(err)
		}
		a.PackageFile = path
		version = proof["resolved_version"].(string)
	} else if a.Provider == "apt-get" {
		policy, _, err := run("RESOLVE_VERSION", "apt-cache", "policy", a.Package)
		if err != nil {
			return fail(err)
		}
		version, err = linuxpkg.APTCandidate(policy)
		if err != nil {
			return fail(err)
		}
		r["expected_version"] = version
		r["resolved_version"] = version
		metadata, _, err := run("RESOLVE_VERSION", "apt-cache", "show", a.Package+"="+version)
		if err != nil {
			return fail(err)
		}
		expectedHash := ""
		for _, line := range strings.Split(metadata, "\n") {
			if strings.HasPrefix(line, "SHA256: ") {
				expectedHash = strings.TrimSpace(strings.TrimPrefix(line, "SHA256: "))
				break
			}
		}
		if len(expectedHash) != 64 {
			return fail(fmt.Errorf("APT metadata contains no SHA256"))
		}
		uris, _, err := run("RESOLVE_DOWNLOAD_SOURCE", "apt-get", "--print-uris", "download", a.Package+"="+version)
		if err != nil {
			return fail(err)
		}
		match := regexp.MustCompile(`(?m)^'([^']+)'`).FindStringSubmatch(uris)
		if len(match) != 2 {
			return fail(fmt.Errorf("APT did not produce an exact HTTP package URL"))
		}
		match[1], err = linuxpkg.ResolveAPTDownload(match[1], os.ReadFile)
		if err != nil {
			return fail(err)
		}
		r["resolved_download_url"] = match[1]
		r["phase"] = "DOWNLOAD_REAL_FILE"
		save()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, match[1], nil)
		if err != nil {
			return fail(err)
		}
		client := &http.Client{Timeout: 15 * time.Minute}
		response, err := client.Do(request)
		if err != nil {
			return fail(err)
		}
		r["download_http_status"] = response.StatusCode
		r["final_download_url"] = response.Request.URL.String()
		r["content_type"] = response.Header.Get("Content-Type")
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return fail(fmt.Errorf("download HTTP %d", response.StatusCode))
		}
		path := filepath.Join(work, "package.deb")
		f, err := os.Create(path)
		if err != nil {
			response.Body.Close()
			return fail(err)
		}
		hash := sha256.New()
		size, err := io.Copy(io.MultiWriter(f, hash), response.Body)
		response.Body.Close()
		f.Close()
		r["downloaded_bytes"] = size
		r["sha256"] = hex.EncodeToString(hash.Sum(nil))
		if err != nil {
			return fail(err)
		}
		if size == 0 || r["sha256"] != expectedHash {
			return fail(fmt.Errorf("download size/hash verification failed"))
		}
		fields, _, err := run("VERIFY_FILE", "dpkg-deb", "-f", path, "Package", "Version")
		if err != nil {
			return fail(err)
		}
		if !strings.Contains(fields, "Package: "+a.Package) || !strings.Contains(fields, "Version: "+version) {
			return fail(fmt.Errorf("downloaded package identity/version mismatch"))
		}
		r["file_validation"] = true
		r["download_real"] = true
	} else if a.Provider == "flatpak" {
		ref, _, err := run("RESOLVE_REF", "flatpak", "remote-info", "--user", "--show-ref", a.FlatpakRemote, a.Package)
		if err != nil {
			return fail(err)
		}
		refParts := strings.Split(strings.TrimSpace(ref), "/")
		if len(refParts) != 4 || refParts[0] != "app" || refParts[1] != a.Package {
			return fail(fmt.Errorf("Flatpak exact application ref not resolved"))
		}
		r["resolved_ref"] = strings.TrimSpace(ref)
		commit, _, err := run("RESOLVE_VERSION", "flatpak", "remote-info", "--user", "--show-commit", a.FlatpakRemote, a.Package)
		if err != nil {
			return fail(err)
		}
		r["expected_version"] = strings.TrimSpace(commit)
		r["resolved_version"] = strings.TrimSpace(commit)
		remotes, _, err := run("RESOLVE_DOWNLOAD_SOURCE", "flatpak", "remotes", "--user", "--columns=name,url")
		if err != nil {
			return fail(err)
		}
		for _, line := range strings.Split(remotes, "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == a.FlatpakRemote {
				r["resolved_download_url"] = fields[1]
			}
		}
		metadata, _, err := run("RESOLVE_EXTRA_DATA", "flatpak", "remote-info", "--user", "--show-metadata", a.FlatpakRemote, a.Package)
		if err != nil {
			return fail(err)
		}
		extraTransfers, err := verifyFlatpakExtraData(ctx, metadata, work)
		r["extra_data_transfers"] = extraTransfers
		if err != nil {
			return fail(err)
		}
		origin, _ := r["resolved_download_url"].(string)
		recorder, err := startPhysicalHTTPRecorder(origin)
		if err != nil {
			return fail(err)
		}
		defer recorder.close()
		defer func() { r["http_transfers"] = recorder.snapshot() }()
		_, _, err = run("ENABLE_HTTP_EVIDENCE", "flatpak", "remote-modify", "--user", "--url="+recorder.endpoint, a.FlatpakRemote)
		if err != nil {
			return fail(err)
		}
		defer run("RESTORE_ORIGINAL_REPOSITORY", "flatpak", "remote-modify", "--user", "--url="+origin, a.FlatpakRemote)
		_, _, err = run("DOWNLOAD_REAL_FILE", "flatpak", "install", "--user", "--no-deploy", "--noninteractive", "--ostree-verbose", "-y", a.FlatpakRemote, a.Package)
		if err != nil {
			return fail(err)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return fail(err)
		}
		bundle := filepath.Join(work, "application.flatpak")
		_, _, err = run("VERIFY_FILE", "flatpak", "build-bundle", "--arch="+refParts[2], filepath.Join(home, ".local/share/flatpak/repo"), bundle, a.Package, refParts[3])
		if err != nil {
			return fail(err)
		}
		f, err := os.Open(bundle)
		if err != nil {
			return fail(err)
		}
		hash := sha256.New()
		size, err := io.Copy(hash, f)
		f.Close()
		if err != nil {
			return fail(err)
		}
		if size == 0 {
			return fail(fmt.Errorf("empty Flatpak bundle"))
		}
		r["downloaded_bytes"] = size
		r["sha256"] = hex.EncodeToString(hash.Sum(nil))
		r["file_validation"] = true
		transfers := recorder.snapshot()
		r["http_transfers"] = transfers
		payloads := 0
		for _, transfer := range transfers {
			if transfer.Status == 200 && transfer.Bytes > 0 && transfer.Error == "" && (strings.Contains(transfer.URL, "/deltas/") || strings.Contains(transfer.URL, "/objects/")) {
				payloads++
			}
		}
		if payloads == 0 {
			return fail(fmt.Errorf("Flatpak package built but no completed repository payload HTTP responses were recorded"))
		}
		r["download_real"] = true
		r["download_http_status"] = http.StatusOK
		r["download_http_status_note"] = "Actual HTTPS repository payload transfers recorded separately; sha256 identifies the verified reconstructed OSTree bundle; vendor extra-data checked separately"
	} else {
		return fail(fmt.Errorf("no download validation adapter for provider %s on this test image", a.Provider))
	}
	spec, err := operationSpec(a, version, false)
	if err != nil {
		return fail(err)
	}
	r["install_command_type"] = spec.Name
	execute := func(phase string, s linuxpkg.Spec) (string, int, error) {
		if s.NeedsRoot {
			return run(phase, "sudo", append([]string{"-n", "--", s.Name}, s.Args...)...)
		}
		return run(phase, s.Name, s.Args...)
	}
	_, code, err := execute("INSTALL_REAL_APPLICATION", spec)
	r["install_exit_code"] = code
	if err != nil {
		return fail(err)
	}
	r["install_real"] = true
	if err = verifyInstalledState(ctx, a, true); err != nil {
		return fail(err)
	}
	r["wiz4rd_detection"] = true
	probe, err := linuxpkg.Detect(a.Provider, a.Package, a.FlatpakRemote)
	if a.Provider == "ollama-vendor" {
		probe = linuxpkg.Spec{Name: "systemctl", Args: []string{"is-active", "ollama.service"}}
		err = nil
	}
	if err != nil {
		return fail(err)
	}
	observed, _, err := run("INDEPENDENT_VERIFY_INSTALLED", probe.Name, probe.Args...)
	if err != nil {
		return fail(err)
	}
	if !linuxpkg.DetectionSuccess(a.Provider, []byte(observed), true) {
		return fail(fmt.Errorf("independent installed state not proven"))
	}
	r["independent_detection"] = true
	var binaries []string
	if a.Provider == "ollama-vendor" {
		info, err := os.Stat(ollamaBinary)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			return fail(fmt.Errorf("Ollama main executable missing"))
		}
		actual, _, err := run("INDEPENDENT_VERIFY_OLLAMA_VERSION", ollamaBinary, "--version")
		if err != nil || !strings.Contains(actual, version) {
			return fail(fmt.Errorf("Ollama binary/service version mismatch: %s", actual))
		}
		binaries = append(binaries, ollamaBinary, ollamaLibraries, ollamaUnit)
	} else if a.Provider == "uv-python" {
		path := strings.TrimSpace(observed)
		info, err := os.Stat(path)
		if err != nil || !filepath.IsAbs(path) || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			return fail(fmt.Errorf("managed Python executable not independently present"))
		}
		actual, _, err := run("INDEPENDENT_VERIFY_PYTHON_VERSION", path, "-I", "-c", "import sys; print('.'.join(map(str, sys.version_info[:3])))")
		if err != nil || strings.TrimSpace(actual) != version {
			return fail(fmt.Errorf("installed Python executable version mismatch: %s", actual))
		}
		binaries = append(binaries, path)
	} else if a.Provider == "apt-get" {
		files, _, err := run("VERIFY_EXECUTABLES", "dpkg-query", "-L", a.Package)
		if err != nil {
			return fail(err)
		}
		for _, path := range strings.Split(files, "\n") {
			if strings.HasPrefix(path, "/usr/") || strings.HasPrefix(path, "/opt/") || strings.HasPrefix(path, "/bin/") || strings.HasPrefix(path, "/sbin/") {
				info, e := os.Stat(path)
				if e == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
					binaries = append(binaries, path)
				}
			}
		}
	} else {
		location, _, err := run("VERIFY_EXECUTABLES", "flatpak", "info", "--user", "--show-location", a.Package)
		if err != nil {
			return fail(err)
		}
		deployment := strings.TrimSpace(location)
		metadata, _, err := run("VERIFY_EXECUTABLE_METADATA", "flatpak", "info", "--user", "--show-metadata", a.Package)
		if err != nil {
			return fail(err)
		}
		application := false
		command := ""
		for _, line := range strings.Split(metadata, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") {
				application = line == "[Application]"
				continue
			}
			if application && strings.HasPrefix(line, "command=") {
				command = strings.TrimPrefix(line, "command=")
			}
		}
		var executable string
		if strings.HasPrefix(command, "/app/") {
			executable = filepath.Join(deployment, "files", strings.TrimPrefix(command, "/app/"))
		} else if command != "" && !strings.Contains(command, "/") {
			executable = filepath.Join(deployment, "files", "bin", command)
		}
		if !filepath.IsAbs(deployment) || executable == "" {
			return fail(fmt.Errorf("Flatpak application command/deployment not independently resolved"))
		}
		// Absolute /app symlinks only resolve inside the installed sandbox.
		// Probe the declared command there, then inspect its actual deployment file.
		sandboxCommand := "/app/bin/" + command
		if strings.HasPrefix(command, "/app/") {
			sandboxCommand = command
		}
		resolved, _, err := run("VERIFY_SANDBOX_EXECUTABLE", "flatpak", "run", "--user", "--command=sh", a.Package,
			"-c", `test -f "$1" && test -x "$1" && readlink -f "$1"`, "physical-probe", sandboxCommand)
		if err != nil {
			return fail(fmt.Errorf("Flatpak declared command verification: %w", err))
		}
		resolved = strings.TrimSpace(resolved)
		if !strings.HasPrefix(resolved, "/app/") || filepath.Clean(resolved) != resolved {
			return fail(fmt.Errorf("Flatpak command does not resolve to an application-owned executable: %q", resolved))
		}
		r["sandbox_executable_path"] = resolved
		executable = filepath.Join(deployment, "files", strings.TrimPrefix(resolved, "/app/"))
		info, err := os.Stat(executable)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			return fail(fmt.Errorf("Flatpak declared executable not present: %s", executable))
		}
		binaries = append(binaries, deployment, executable)
		installedCommit, _, err := run("VERIFY_VERSION", "flatpak", "info", "--user", "--show-commit", a.Package)
		if err != nil {
			return fail(err)
		}
		if strings.TrimSpace(installedCommit) != r["resolved_version"] {
			return fail(fmt.Errorf("Flatpak commit changed between download and install"))
		}
	}
	r["installed_binary_paths"] = binaries
	spec, err = operationSpec(a, version, true)
	if err != nil {
		return fail(err)
	}
	_, code, err = execute("UNINSTALL_THROUGH_WIZ4RDFR0G", spec)
	r["uninstall_exit_code"] = code
	if err != nil {
		return fail(err)
	}
	r["uninstall_real"] = true
	if err = verifyInstalledState(ctx, a, false); err != nil {
		return fail(err)
	}
	r["wiz4rd_removed_detection"] = true
	observed, code, err = run("INDEPENDENT_VERIFY_REMOVED", probe.Name, probe.Args...)
	if a.Provider == "ollama-vendor" {
		state, exitCode, stateErr := run("INDEPENDENT_VERIFY_SERVICE_REMOVED", "systemctl", "show", "-p", "LoadState", "--value", "ollama.service")
		if stateErr != nil || exitCode != 0 || strings.TrimSpace(state) != "not-found" {
			return fail(fmt.Errorf("Ollama systemd registration still present: %s", state))
		}
		_, accountCode, _ := run("INDEPENDENT_VERIFY_SERVICE_ACCOUNT_REMOVED", "getent", "passwd", "ollama")
		if accountCode != 2 {
			return fail(fmt.Errorf("Ollama service account absence not proven"))
		}
	} else if a.Provider == "uv-python" {
		inventory, _ := linuxpkg.Inventory(a.Provider)
		output, exitCode, inventoryErr := run("INDEPENDENT_VERIFY_MANAGED_REGISTRATION_REMOVED", inventory.Name, inventory.Args...)
		present, parseErr := linuxpkg.InventoryContains(a.Provider, a.Package, []byte(output), exitCode)
		if inventoryErr != nil || parseErr != nil || present {
			return fail(fmt.Errorf("managed Python registration removal not proven"))
		}
	} else if a.Provider == "apt-get" {
		if !linuxpkg.DPKGRemoved(observed, code) {
			return fail(fmt.Errorf("independent dpkg removal not proven: exit %d", code))
		}
	} else {
		if code != 1 || !strings.Contains(strings.ToLower(observed), "not installed") {
			return fail(fmt.Errorf("independent Flatpak removal not proven: exit %d", code))
		}
	}
	for _, path := range binaries {
		if _, err = os.Stat(path); !os.IsNotExist(err) {
			return fail(fmt.Errorf("application binary/deployment still present or inaccessible: %s", path))
		}
	}
	r["independent_removed_detection"] = true
	if len(binaries) == 0 {
		return fail(fmt.Errorf("lifecycle executed but no application executable path was independently identified"))
	}
	r["failure_reason"] = ""
	r["final_status"] = "FULL_PASS"
	return 0
}

func linuxLifecycleArgs() int {
	index, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return linuxLifecycle(index, os.Args[3])
}
