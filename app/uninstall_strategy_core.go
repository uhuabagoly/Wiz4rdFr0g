package main

import (
	"strings"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

type uninstallFacts struct {
	PackageID        string
	Scope            string
	WindowsInstaller bool
	MSIProductCode   bool
	HasUninstall     bool
}

func decideUninstallStrategy(profile catalogpkg.Profile, facts uninstallFacts) catalogpkg.UninstallStrategy {
	if profile.SystemComponent {
		return catalogpkg.StrategySystemComponentUnsupported
	}
	if profile.UninstallStrategy == catalogpkg.StrategyManualOnly {
		return catalogpkg.StrategyManualOnly
	}
	if profile.InstallerType == catalogpkg.InstallerMSIX {
		return catalogpkg.StrategyMSIXStore
	}
	if facts.WindowsInstaller || facts.MSIProductCode {
		return catalogpkg.StrategyMSIProductCode
	}
	if strings.TrimSpace(facts.PackageID) != "" {
		switch strings.ToLower(strings.TrimSpace(facts.Scope)) {
		case "user":
			return catalogpkg.StrategyWingetUser
		case "machine":
			return catalogpkg.StrategyWingetMachine
		default:
			return catalogpkg.StrategyRuntimeDetect
		}
	}
	if facts.HasUninstall {
		switch strings.ToLower(strings.TrimSpace(facts.Scope)) {
		case "user":
			return catalogpkg.StrategyRegistryUser
		case "machine":
			return catalogpkg.StrategyRegistryMachine
		default:
			return catalogpkg.StrategyVendorUninstaller
		}
	}
	return catalogpkg.StrategyUnresolved
}

func strategyAllowsAutomaticUninstall(strategy catalogpkg.UninstallStrategy) bool {
	switch strategy {
	case catalogpkg.StrategySystemComponentUnsupported, catalogpkg.StrategyManualOnly, catalogpkg.StrategyUnresolved:
		return false
	default:
		return true
	}
}
