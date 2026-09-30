//go:build windows

package main

import "strings"

// v1.16.4's MSI defaults REMOVE_DATA to yes even in silent mode. The publisher
// explicitly conditions both instance purge and data deletion on the value yes.
// Bind the tested MSI identity before selecting the documented no value.
// https://github.com/canonical/multipass/blob/v1.16.4/packaging/windows/wix/Package.wxs
func multipassPreserveDataRegistration(reg registryPackage) bool {
	const guid = "{A7AD2F65-C450-4440-9AD9-59591C1AA15E}"
	return reg.DisplayName == "Multipass" && reg.DisplayVersion == "1.16.4" && reg.Scope == "machine" && reg.WindowsInstaller == 1 && reg.RegistryView == "/reg:64" &&
		strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+guid) &&
		strings.EqualFold(registeredMSIProductCode(reg.UninstallString, reg.QuietUninstallString, reg.RegistryKey, true), guid)
}
