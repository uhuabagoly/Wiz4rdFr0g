//go:build windows

package main

import (
	"strings"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func installedUninstallStrategy(app appDef, pkg installedPackage) catalogpkg.UninstallStrategy {
	profile := catalogpkg.ProfileFor(app)
	if strings.EqualFold(strings.TrimSpace(pkg.Source), "msstore") {
		profile.InstallerType = catalogpkg.InstallerMSIX
	}
	facts := uninstallFacts{
		PackageID:        pkg.ID,
		Scope:            pkg.Scope,
		WindowsInstaller: pkg.WindowsInstaller,
		MSIProductCode:   extractMSIProductCode(pkg.UninstallString+" "+pkg.QuietUninstallString+" "+pkg.RegistryKey) != "",
		HasUninstall:     strings.TrimSpace(pkg.QuietUninstallString) != "" || strings.TrimSpace(pkg.UninstallString) != "",
	}
	return decideUninstallStrategy(profile, facts)
}

func catalogBlocksAutomaticUninstall(app appDef) bool {
	profile := catalogpkg.ProfileFor(app)
	return profile.SystemComponent || profile.UninstallStrategy == catalogpkg.StrategyManualOnly || profile.UninstallStrategy == catalogpkg.StrategyUnresolved
}
