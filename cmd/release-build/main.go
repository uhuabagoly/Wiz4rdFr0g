package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

var generatedPrefixes = []string{
	"audit/",
	"dist/",
	"release/",
	"test/windows-vm/results/",
}

var generatedExact = map[string]bool{
	"dist-linux/Wiz4rdFr0g":            true,
	"dist-linux/SHA256SUMS":            true,
	"installer/payload/Wiz4rdFr0g.exe": true,
	"Wiz4rd_Fr0g_Linux_x64.tar.gz":     true,
}

func main() {
	manifestPath := "release/build_manifest.json"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		manifestPath = os.Args[1]
	}
	if err := buildRelease(manifestPath); err != nil {
		fmt.Fprintln(os.Stderr, "release build failed:", err)
		os.Exit(1)
	}
}

func buildRelease(manifestPath string) error {
	gitCommit, err := resolveGitCommit()
	if err != nil {
		return err
	}
	if dirty, err := sourceTreeDirty(); err != nil {
		return err
	} else if len(dirty) != 0 {
		return fmt.Errorf("source tree contains uncommitted source changes: %s", strings.Join(dirty, ", "))
	}
	fingerprint, err := releaseproof.CurrentCatalogFingerprint()
	if err != nil {
		return fmt.Errorf("catalog fingerprint: %w", err)
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}
	if err := os.MkdirAll("dist-linux", 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	winApp := filepath.FromSlash("dist/Wiz4rdFr0g.exe")
	ld := "-H windowsgui -X main.buildGitCommit=" + gitCommit
	if err := runEnv([]string{"GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0"}, "go", "build", "-trimpath", "-ldflags="+ld, "-o", winApp, "./app"); err != nil {
		return fmt.Errorf("build Windows app: %w", err)
	}

	payload := filepath.FromSlash("installer/payload/Wiz4rdFr0g.exe")
	if err := copyFile(winApp, payload, 0755); err != nil {
		return fmt.Errorf("copy installer payload: %w", err)
	}
	appHash, appSize, err := releaseproof.FileSHA256(winApp)
	if err != nil {
		return err
	}
	payloadHash, payloadSize, err := releaseproof.FileSHA256(payload)
	if err != nil {
		return err
	}
	payloadConsistent := strings.EqualFold(appHash, payloadHash) && appSize == payloadSize
	if !payloadConsistent {
		return errors.New("installer payload hash differs from Windows application artifact")
	}

	winSetup := filepath.FromSlash("dist/Wiz4rd_Fr0g_Setup.exe")
	if err := runEnv([]string{"GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0"}, "go", "build", "-trimpath", "-ldflags=-H windowsgui", "-o", winSetup, "./installer"); err != nil {
		return fmt.Errorf("build Windows setup: %w", err)
	}

	linuxApp := filepath.FromSlash("dist-linux/Wiz4rdFr0g")
	if err := runEnv(nil, "go", "build", "-trimpath", "-o", linuxApp, "./linux"); err != nil {
		return fmt.Errorf("build Linux app: %w", err)
	}

	linuxPackage := "Wiz4rd_Fr0g_Linux_x64.tar.gz"
	packageFiles := []string{
		filepath.FromSlash("dist-linux/Wiz4rdFr0g"),
		filepath.FromSlash("dist-linux/Wiz4rdFr0g_icon.png"),
		filepath.FromSlash("dist-linux/install.sh"),
		filepath.FromSlash("dist-linux/uninstall.sh"),
		filepath.FromSlash("dist-linux/SHA256SUMS"),
	}
	if err := createTarGz(linuxPackage, packageFiles); err != nil {
		return fmt.Errorf("build Linux package: %w", err)
	}

	setupHash, setupSize, err := releaseproof.FileSHA256(winSetup)
	if err != nil {
		return err
	}
	linuxHash, linuxSize, err := releaseproof.FileSHA256(linuxApp)
	if err != nil {
		return err
	}
	packageHash, packageSize, err := releaseproof.FileSHA256(linuxPackage)
	if err != nil {
		return err
	}

	manifest := releaseproof.BuildManifest{
		SchemaVersion:         releaseproof.BuildManifestSchemaVersion,
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339Nano),
		AppVersion:            releaseproof.AppVersion,
		GitCommit:             gitCommit,
		CatalogFingerprint:    fingerprint,
		EvidenceSchemaVersion: releaseproof.EvidenceSchemaVersion,
		Artifacts: map[string]releaseproof.Artifact{
			releaseproof.PrimaryWindowsArtifactKey:   {Path: filepath.ToSlash(winApp), SHA256: appHash, Size: appSize},
			releaseproof.WindowsSetupArtifactKey:     {Path: filepath.ToSlash(winSetup), SHA256: setupHash, Size: setupSize},
			releaseproof.InstallerPayloadArtifactKey: {Path: filepath.ToSlash(payload), SHA256: payloadHash, Size: payloadSize},
			releaseproof.LinuxAppArtifactKey:         {Path: filepath.ToSlash(linuxApp), SHA256: linuxHash, Size: linuxSize},
			releaseproof.LinuxPackageArtifactKey:     {Path: filepath.ToSlash(linuxPackage), SHA256: packageHash, Size: packageSize},
		},
		PayloadConsistent: payloadConsistent,
	}
	manifest.BuildID = releaseproof.BuildID(manifest.AppVersion, manifest.GitCommit, manifest.CatalogFingerprint, appHash, manifest.EvidenceSchemaVersion)

	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	tmp := manifestPath + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, manifestPath); err != nil {
		return err
	}
	fmt.Printf("build_id=%s git=%s catalog=%s windows_app_sha256=%s payload_consistent=%v\n", manifest.BuildID, manifest.GitCommit, manifest.CatalogFingerprint, appHash, payloadConsistent)
	return nil
}

func resolveGitCommit() (string, error) {
	if v := strings.TrimSpace(os.Getenv("WIZ4RDFR0G_GIT_COMMIT")); v != "" {
		if !releaseproof.ValidGitCommit(v) {
			return "", errors.New("WIZ4RDFR0G_GIT_COMMIT must be a 40-hex Git commit SHA")
		}
		return strings.ToLower(v), nil
	}
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", errors.New("Git commit unavailable; build from a Git checkout or set WIZ4RDFR0G_GIT_COMMIT")
	}
	v := strings.TrimSpace(string(out))
	if !releaseproof.ValidGitCommit(v) {
		return "", errors.New("git rev-parse HEAD did not return a 40-hex commit SHA")
	}
	return strings.ToLower(v), nil
}

func sourceTreeDirty() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	var dirty []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if i := strings.LastIndex(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		path = filepath.ToSlash(strings.Trim(path, "\""))
		if isGenerated(path) {
			continue
		}
		dirty = append(dirty, path)
	}
	sort.Strings(dirty)
	return dirty, nil
}

func isGenerated(path string) bool {
	if generatedExact[path] {
		return true
	}
	for _, p := range generatedPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func runEnv(extra []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), extra...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(src, dst string, mode os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, mode)
}

func createTarGz(outPath string, files []string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	closeAll := func() error {
		if err := tw.Close(); err != nil {
			_ = gz.Close()
			_ = f.Close()
			return err
		}
		if err := gz.Close(); err != nil {
			_ = f.Close()
			return err
		}
		return f.Close()
	}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			return err
		}
		hdr.Name = filepath.ToSlash(filepath.Join("Wiz4rd_Fr0g_Linux_x64", filepath.Base(path)))
		// Do not make the package byte stream depend on filesystem mtimes.
		hdr.ModTime = time.Unix(0, 0).UTC()
		hdr.AccessTime = time.Time{}
		hdr.ChangeTime = time.Time{}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			return err
		}
		_, copyErr := io.Copy(tw, in)
		closeErr := in.Close()
		if copyErr != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			return copyErr
		}
		if closeErr != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			return closeErr
		}
	}
	return closeAll()
}
