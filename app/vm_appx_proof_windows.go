//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

func vmReadAppxFamily(ctx context.Context, family string) ([]releaseproof.AppxProof, error) {
	// Family comes exclusively from the publisher-bound allowlist above.
	if family != releaseproof.ExpectedAppxFamily("Proton.ProtonPass") && family != releaseproof.ExpectedAppxFamily("M2Team.NanaZip") {
		return nil, fmt.Errorf("unreviewed Appx identity")
	}
	script := `[Console]::OutputEncoding=[Text.UTF8Encoding]::new(); $ErrorActionPreference='Stop'; $items=@(Get-AppxPackage | Where-Object { $_.PackageFamilyName -ceq '` + family + `' } | ForEach-Object { $p=$_; $m=Get-AppxPackageManifest -Package $p.PackageFullName; [pscustomobject]@{family=$p.PackageFamilyName;full_name=$p.PackageFullName;install_location=$p.InstallLocation;executables=@($m.Package.Applications.Application | ForEach-Object { [string]$_.Executable })} }); ConvertTo-Json -InputObject $items -Depth 5 -Compress`
	code, out, err := runDirectProcess(ctx, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script})
	if err != nil || code != 0 {
		return nil, fmt.Errorf("Appx registration query failed: %s", compactLog(out))
	}
	var items []releaseproof.AppxProof
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(out), "\ufeff")), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func vmCaptureAppx(ctx context.Context, name string) (vmFilesystemProof, error) {
	var proof vmFilesystemProof
	id := ""
	switch name {
	case "Proton Pass":
		id = "Proton.ProtonPass"
	case "NanaZip":
		id = "M2Team.NanaZip"
	}
	family := releaseproof.ExpectedAppxFamily(id)
	if family == "" {
		return proof, fmt.Errorf("no unambiguous independent uninstall registration")
	}
	items, err := vmReadAppxFamily(ctx, family)
	if err != nil {
		return proof, err
	}
	if len(items) != 1 || items[0].Family != family || items[0].FullName == "" {
		return proof, fmt.Errorf("Appx deployment is absent or ambiguous")
	}
	appx := items[0]
	if !filepath.IsAbs(appx.InstallLocation) {
		return proof, fmt.Errorf("Appx deployment location is not absolute")
	}
	for _, entry := range appx.Executables {
		if entry == "" || filepath.IsAbs(entry) || !strings.EqualFold(filepath.Ext(entry), ".exe") {
			continue
		}
		path := filepath.Join(appx.InstallLocation, entry)
		relative, err := filepath.Rel(appx.InstallLocation, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, `..\`) {
			return proof, fmt.Errorf("Appx executable escapes deployment")
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return proof, fmt.Errorf("Appx manifest executable missing: %s", path)
		}
		proof.BinaryPaths = append(proof.BinaryPaths, path)
	}
	if len(proof.BinaryPaths) == 0 {
		return proof, fmt.Errorf("Appx manifest has no observed executable")
	}
	appx.Present = true
	proof.Appx = &appx
	proof.BinariesPresent = true
	return proof, nil
}
