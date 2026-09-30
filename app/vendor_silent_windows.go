//go:build windows

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Documented vendor modes for physically observed registrations. Do not infer
// an installer framework from an arbitrary uninstall.exe filename.
// Qt: https://doc.qt.io/qtinstallerframework/ifw-use-cases-cli.html
// install4j: https://www.ej-technologies.com/resources/install4j/help/doc/installers/installerModes.html
func observedVendorSilentArgs(app appDef, reg registryPackage) ([]string, bool) {
	if reg.WindowsInstaller != 0 {
		return nil, false
	}
	exe, args, err := splitRegisteredCommandRaw(reg.UninstallString)
	root := filepath.Clean(strings.Trim(reg.InstallLocation, `"`))
	// The publisher documents -unat for the client instance BvSshClient.
	// https://bitvise.com/ssh-server-guide-installing
	if app.Name == "Bitvise SSH Client" && err == nil && reg.DisplayName == "Bitvise SSH Client 9.66 (remove only)" && reg.DisplayVersion == "9.66" && reg.Scope == "machine" && reg.RegistryView == "/reg:32" && len(args) == 1 && args[0] == "BvSshClient" {
		programFiles := os.Getenv("ProgramFiles(x86)")
		if filepath.IsAbs(programFiles) && strings.EqualFold(exe, filepath.Join(programFiles, "Bitvise SSH Client", "uninst.exe")) && strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\BvSshClient`) {
			return []string{"BvSshClient", "-unat"}, true
		}
	}
	// Chromium's --force-uninstall is its documented silent mode; profile
	// deletion is a separate switch, deliberately never supplied here.
	// https://github.com/chromium/chromium/blob/main/chrome/installer/setup/uninstall.cc
	if (app.Name == "Brave" || app.Name == "Chromium") && err == nil && reg.DisplayName == app.Name && reg.Scope == "user" && len(args) == 1 && args[0] == "--uninstall" && regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(reg.DisplayVersion) {
		name, relative := "Chromium", filepath.Join("Chromium", "Application")
		if app.Name == "Brave" {
			name, relative = "BraveSoftware Brave-Browser", filepath.Join("BraveSoftware", "Brave-Browser", "Application")
		}
		local := os.Getenv("LOCALAPPDATA")
		key := `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + name
		if filepath.IsAbs(local) && strings.EqualFold(root, filepath.Join(local, relative)) && strings.EqualFold(reg.RegistryKey, key) && strings.EqualFold(exe, filepath.Join(root, reg.DisplayVersion, "Installer", "setup.exe")) {
			return []string{"--uninstall", "--force-uninstall"}, true
		}
	}
	if err != nil || !filepath.IsAbs(root) || !strings.EqualFold(filepath.Dir(exe), root) {
		return nil, false
	}
	switch app.Name {
	case "Windscribe":
		// v2.24.13 uninstaller/main.cpp recognizes /VERYSILENT, not /SILENT.
		if reg.DisplayName == "Windscribe" && reg.DisplayVersion == "2.24.13" && reg.Scope == "machine" && reg.RegistryView == "/reg:64" && len(args) == 0 && strings.EqualFold(filepath.Base(exe), "uninstall.exe") && strings.EqualFold(root, filepath.Join(os.Getenv("ProgramFiles"), "Windscribe")) && strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{fa690e90-ddb0-4f0c-b3f1-136c084e5fc7}_is1`) {
			return []string{"/VERYSILENT"}, true
		}
	case "Apache NetBeans":
		// Apache release250 nbi/engine/.../cli/options/SilentOption.java.
		if reg.DisplayName == "Apache NetBeans IDE 25" && reg.DisplayVersion == "25" && reg.Scope == "machine" && reg.QuietUninstallString == "" && len(args) == 0 && strings.EqualFold(filepath.Base(exe), "uninstall.exe") && strings.EqualFold(root, filepath.Join(os.Getenv("ProgramFiles"), "NetBeans-25")) && strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\nbi-nb-all-25.0.0.250214.0`) {
			return []string{"--silent"}, true
		}
	case "mitmproxy":
		// Publisher installer project: release/installbuilder/mitmproxy.xml.
		// InstallBuilder documents --mode unattended for its uninstaller too.
		if reg.DisplayName == "mitmproxy" && reg.DisplayVersion == "12.2.3" && reg.Scope == "machine" && reg.QuietUninstallString == "" && len(args) == 0 && strings.EqualFold(filepath.Base(exe), "uninstall.exe") && strings.EqualFold(root, filepath.Join(os.Getenv("ProgramFiles"), "mitmproxy")) && strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\mitmproxy 12.2.3`) {
			return []string{"--mode", "unattended"}, true
		}
	case "GPT4All":
		if reg.DisplayName == "GPT4All" && reg.DisplayVersion == "3.10.0" && reg.Scope == "user" && strings.EqualFold(filepath.Base(exe), "maintenancetool.exe") && len(args) == 1 && args[0] == "--start-uninstaller" {
			return []string{"--confirm-command", "--default-answer", "purge"}, true
		}
	case "SoapUI", "BiglyBT":
		identity := app.Name == "SoapUI" && reg.DisplayName == "SoapUI 5.10.0" && reg.DisplayVersion == "5.10.0" && filepath.Base(reg.RegistryKey) == "5517-2803-0637-4585"
		identity = identity || app.Name == "BiglyBT" && reg.DisplayName == "BiglyBT" && reg.DisplayVersion == "4.1.0.0" && filepath.Base(reg.RegistryKey) == "0112-2557-8304-7048"
		if identity && reg.Scope == "machine" && strings.EqualFold(filepath.Base(exe), "uninstall.exe") && len(args) == 0 {
			if info, err := os.Stat(filepath.Join(root, ".install4j")); err == nil && info.IsDir() {
				return []string{"-q"}, true
			}
		}
	}
	return nil, false
}
