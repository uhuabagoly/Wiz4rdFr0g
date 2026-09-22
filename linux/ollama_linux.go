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
	archivepath "path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const ollamaBinary = "/usr/local/bin/ollama"
const ollamaLibraries = "/usr/local/lib/ollama"
const ollamaUnit = "/etc/systemd/system/ollama.service"
const ollamaOwnership = "/var/lib/wiz4rdfr0g/ollama.json"

type ollamaAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

func ollamaRelease(ctx context.Context, version string) (ollamaAsset, string, error) {
	endpoint := "https://api.github.com/repos/ollama/ollama/releases/latest"
	if version != "" && version != "Legújabb" {
		version = strings.TrimPrefix(version, "v")
		if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(version) {
			return ollamaAsset{}, "", fmt.Errorf("invalid stable Ollama version")
		}
		endpoint = "https://api.github.com/repos/ollama/ollama/releases/tags/v" + version
	}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return ollamaAsset{}, "", err
	}
	resp, err := (&http.Client{Timeout: time.Minute}).Do(req)
	if err != nil {
		return ollamaAsset{}, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ollamaAsset{}, "", fmt.Errorf("Ollama release API HTTP %d", resp.StatusCode)
	}
	var release struct {
		Tag        string        `json:"tag_name"`
		Prerelease bool          `json:"prerelease"`
		Assets     []ollamaAsset `json:"assets"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&release); err != nil {
		return ollamaAsset{}, "", err
	}
	if release.Prerelease || !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(release.Tag) {
		return ollamaAsset{}, "", fmt.Errorf("Ollama stable release not resolved")
	}
	arch := map[string]string{"amd64": "amd64", "arm64": "arm64"}[runtime.GOARCH]
	for _, asset := range release.Assets {
		if arch != "" && asset.Name == "ollama-linux-"+arch+".tar.zst" && asset.URL == "https://github.com/ollama/ollama/releases/download/"+release.Tag+"/"+asset.Name && regexp.MustCompile(`^sha256:[a-f0-9]{64}$`).MatchString(asset.Digest) && asset.Size > 0 {
			return asset, strings.TrimPrefix(release.Tag, "v"), nil
		}
	}
	return ollamaAsset{}, "", fmt.Errorf("official Ollama platform asset/checksum missing")
}

func downloadOllama(ctx context.Context, directory, version string) (string, map[string]any, error) {
	proof := map[string]any{}
	asset, resolved, err := ollamaRelease(ctx, version)
	if err != nil {
		return "", proof, err
	}
	proof["resolved_download_url"], proof["resolved_version"], proof["expected_version"], proof["expected_sha256"] = asset.URL, resolved, resolved, strings.TrimPrefix(asset.Digest, "sha256:")
	req, err := http.NewRequestWithContext(ctx, "GET", asset.URL, nil)
	if err != nil {
		return "", proof, err
	}
	resp, err := (&http.Client{Timeout: 25 * time.Minute}).Do(req)
	if err != nil {
		return "", proof, err
	}
	defer resp.Body.Close()
	proof["download_http_status"], proof["final_download_url"], proof["content_type"] = resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type")
	if resp.StatusCode != 200 {
		return "", proof, fmt.Errorf("Ollama payload HTTP %d", resp.StatusCode)
	}
	path := filepath.Join(directory, "ollama.tar.zst")
	f, err := os.Create(path)
	if err != nil {
		return "", proof, err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(f, hash), io.LimitReader(resp.Body, asset.Size+1))
	f.Close()
	actual := hex.EncodeToString(hash.Sum(nil))
	proof["downloaded_bytes"], proof["sha256"] = size, actual
	if err != nil || size != asset.Size || "sha256:"+actual != asset.Digest {
		return "", proof, fmt.Errorf("Ollama release asset size/hash mismatch: %v", err)
	}
	if err = validateOllamaArchive(ctx, path); err != nil {
		return "", proof, err
	}
	proof["file_validation"], proof["download_real"] = true, true
	return path, proof, nil
}

func validateOllamaArchive(ctx context.Context, path string) error {
	out, err := exec.CommandContext(ctx, "tar", "--zstd", "-tf", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Ollama archive validation: %w: %s", err, out)
	}
	return validateOllamaPaths(string(out))
}

func validateOllamaPaths(listing string) error {
	found := false
	for _, line := range strings.Split(strings.TrimSpace(listing), "\n") {
		name := strings.TrimSuffix(strings.TrimPrefix(line, "./"), "/")
		if name == "bin/ollama" {
			found = true
		}
		if archivepath.Clean(name) != name || (name != "bin" && name != "lib" && name != "bin/ollama" && name != "lib/ollama" && !strings.HasPrefix(name, "lib/ollama/")) {
			return fmt.Errorf("unexpected path in official Ollama archive: %q", line)
		}
	}
	if !found {
		return fmt.Errorf("official Ollama archive has no main executable")
	}
	return nil
}

func ollamaInstalled() (bool, error) {
	present := 0
	for _, path := range []string{ollamaBinary, ollamaLibraries, ollamaUnit, ollamaOwnership} {
		_, err := os.Lstat(path)
		if err == nil {
			present++
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	if present == 0 {
		return false, nil
	}
	if present != 4 {
		return false, fmt.Errorf("partial or externally managed Ollama installation; state requires review")
	}
	return true, nil
}

// Implements the publisher's documented manual installation and service
// teardown: https://docs.ollama.com/linux . Only an installation created by
// this application is removed. Existing model data is retained.
func ollamaWorker(args []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("Ollama system installation requires elevation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	run := func(name string, args ...string) error {
		fmt.Println(name, strings.Join(args, " "))
		out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
		fmt.Print(string(out))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if len(args) == 3 && args[0] == "install" {
		for _, path := range []string{ollamaBinary, ollamaLibraries, ollamaUnit, ollamaOwnership} {
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				return fmt.Errorf("refusing to overwrite existing Ollama path: %s", path)
			}
		}
		if err := exec.CommandContext(ctx, "getent", "passwd", "ollama").Run(); err == nil {
			return fmt.Errorf("pre-existing Ollama service account")
		}
		asset, version, err := ollamaRelease(ctx, args[2])
		if err != nil {
			return err
		}
		f, err := os.Open(args[1])
		if err != nil {
			return err
		}
		defer f.Close()
		hash := sha256.New()
		size, err := io.Copy(hash, f)
		if err != nil || size != asset.Size || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != asset.Digest {
			return fmt.Errorf("privileged installer rejected unauthenticated Ollama package")
		}
		if err = validateOllamaArchive(ctx, args[1]); err != nil {
			return err
		}
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		extract := exec.CommandContext(ctx, "tar", "--zstd", "-xf", "-", "-C", "/usr/local", "--no-same-owner")
		extract.Stdin = f
		out, err := extract.CombinedOutput()
		fmt.Print(string(out))
		if err != nil {
			return err
		}
		if err = run("useradd", "-r", "-s", "/bin/false", "-U", "-m", "-d", "/usr/share/ollama", "ollama"); err != nil {
			return err
		}
		unit := "[Unit]\nDescription=Ollama Service\nAfter=network-online.target\n\n[Service]\nExecStart=" + ollamaBinary + " serve\nUser=ollama\nGroup=ollama\nRestart=always\nRestartSec=3\n\n[Install]\nWantedBy=multi-user.target\n"
		if err = os.WriteFile(ollamaUnit, []byte(unit), 0644); err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(ollamaOwnership), 0755); err != nil {
			return err
		}
		manifest, _ := json.Marshal(map[string]string{"owner": "Wiz4rdFr0g", "version": version, "sha256": strings.TrimPrefix(asset.Digest, "sha256:")})
		if err = os.WriteFile(ollamaOwnership, manifest, 0644); err != nil {
			return err
		}
		if err = run("systemctl", "daemon-reload"); err != nil {
			return err
		}
		if err = run("systemctl", "enable", "--now", "ollama.service"); err != nil {
			return err
		}
		for attempt := 0; attempt < 30; attempt++ {
			resp, err := (&http.Client{Timeout: time.Second}).Get("http://127.0.0.1:11434/api/version")
			if err == nil {
				var result struct {
					Version string `json:"version"`
				}
				decodeErr := json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()
				if decodeErr == nil && result.Version == version {
					return nil
				}
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("installed Ollama service did not report expected version")
	}
	if len(args) == 1 && args[0] == "remove" {
		body, err := os.ReadFile(ollamaOwnership)
		if err != nil {
			return fmt.Errorf("no Wiz4rdFr0g ownership record: %w", err)
		}
		var manifest map[string]string
		if json.Unmarshal(body, &manifest) != nil || manifest["owner"] != "Wiz4rdFr0g" {
			return fmt.Errorf("invalid Ollama ownership record")
		}
		if err = run("systemctl", "stop", "ollama.service"); err != nil {
			return err
		}
		if err = run("systemctl", "disable", "ollama.service"); err != nil {
			return err
		}
		if err = os.Remove(ollamaUnit); err != nil {
			return err
		}
		if err = run("systemctl", "daemon-reload"); err != nil {
			return err
		}
		if err = os.Remove(ollamaBinary); err != nil {
			return err
		}
		if err = os.RemoveAll(ollamaLibraries); err != nil {
			return err
		}
		if err = run("userdel", "ollama"); err != nil {
			return err
		}
		if err = exec.CommandContext(ctx, "getent", "group", "ollama").Run(); err == nil {
			if err = run("groupdel", "ollama"); err != nil {
				return err
			}
		}
		return os.Remove(ollamaOwnership)
	}
	return fmt.Errorf("invalid Ollama vendor worker arguments")
}
