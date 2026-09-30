//go:build windows

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

func vmCaptureSeafileRetention(proof *vmFilesystemProof) error {
	if !seafilePreserveDataRegistration(proof.Registration) || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" {
		return fmt.Errorf("Seafile settings witness requires the bound MSI on a disposable runner")
	}
	var seed [48]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return err
	}
	p := &releaseproof.SeafileRetentionProof{Key: `HKEY_CURRENT_USER\SOFTWARE\Seafile`, View: "Registry64", Name: "Wiz4rdFr0g-retention-" + hex.EncodeToString(seed[:16]), Value: hex.EncodeToString(seed[16:])}
	// Independent .NET registry API, separate from the production native guard.
	script := `$ErrorActionPreference='Stop'; $b=[Microsoft.Win32.RegistryKey]::OpenBaseKey([Microsoft.Win32.RegistryHive]::CurrentUser,[Microsoft.Win32.RegistryView]::Registry64); try { $k=$b.CreateSubKey('SOFTWARE\Seafile'); try { if($null -ne $k.GetValue('` + psQuote(p.Name) + `')){throw 'Witness already exists'}; $k.SetValue('` + psQuote(p.Name) + `','` + p.Value + `',[Microsoft.Win32.RegistryValueKind]::String) } finally {$k.Dispose()} } finally {$b.Dispose()}`
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	code, out, err := runDirectProcess(ctx, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script})
	if err != nil || code != 0 {
		return fmt.Errorf("create Seafile witness: exit=%d %s %v", code, out, err)
	}
	proof.SeafileRetention = p
	return nil
}

func vmVerifySeafileRetention(p *releaseproof.SeafileRetentionProof) error {
	script := `$ErrorActionPreference='Stop'; $b=[Microsoft.Win32.RegistryKey]::OpenBaseKey([Microsoft.Win32.RegistryHive]::CurrentUser,[Microsoft.Win32.RegistryView]::Registry64); try { $k=$b.OpenSubKey('SOFTWARE\Seafile'); if($null -eq $k){throw 'Settings key removed'}; try { if($k.GetValueKind('` + psQuote(p.Name) + `') -ne [Microsoft.Win32.RegistryValueKind]::String -or $k.GetValue('` + psQuote(p.Name) + `') -cne '` + psQuote(p.Value) + `'){throw 'Settings witness changed'}; 'PRESERVED' } finally {$k.Dispose()} } finally {$b.Dispose()}`
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	code, out, err := runDirectProcess(ctx, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", script})
	p.Preserved = err == nil && code == 0 && strings.TrimSpace(out) == "PRESERVED"
	if !p.Preserved {
		return fmt.Errorf("Seafile settings were removed or altered: exit=%d %s %v", code, out, err)
	}
	return nil
}
