package main

import (
	"testing"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func baseProfile() catalogpkg.Profile {
	return catalogpkg.Profile{Platform: catalogpkg.PlatformWindows, UninstallStrategy: catalogpkg.StrategyRuntimeDetect, InstallerType: catalogpkg.InstallerUnknown}
}

func TestUninstallStrategyCoverage(t *testing.T) {
	cases := []struct {
		name    string
		profile catalogpkg.Profile
		facts   uninstallFacts
		want    catalogpkg.UninstallStrategy
	}{
		{"winget-user", baseProfile(), uninstallFacts{PackageID: "A.B", Scope: "user"}, catalogpkg.StrategyWingetUser},
		{"winget-machine", baseProfile(), uninstallFacts{PackageID: "A.B", Scope: "machine"}, catalogpkg.StrategyWingetMachine},
		{"winget-dynamic", baseProfile(), uninstallFacts{PackageID: "A.B"}, catalogpkg.StrategyRuntimeDetect},
		{"registry-user", baseProfile(), uninstallFacts{HasUninstall: true, Scope: "user"}, catalogpkg.StrategyRegistryUser},
		{"registry-machine", baseProfile(), uninstallFacts{HasUninstall: true, Scope: "machine"}, catalogpkg.StrategyRegistryMachine},
		{"vendor", baseProfile(), uninstallFacts{HasUninstall: true}, catalogpkg.StrategyVendorUninstaller},
		{"msi", baseProfile(), uninstallFacts{WindowsInstaller: true}, catalogpkg.StrategyMSIProductCode},
		{"msix", catalogpkg.Profile{InstallerType: catalogpkg.InstallerMSIX}, uninstallFacts{}, catalogpkg.StrategyMSIXStore},
		{"system", catalogpkg.Profile{SystemComponent: true}, uninstallFacts{PackageID: "A.B"}, catalogpkg.StrategySystemComponentUnsupported},
		{"manual", catalogpkg.Profile{UninstallStrategy: catalogpkg.StrategyManualOnly}, uninstallFacts{}, catalogpkg.StrategyManualOnly},
		{"unresolved", baseProfile(), uninstallFacts{}, catalogpkg.StrategyUnresolved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideUninstallStrategy(tc.profile, tc.facts); got != tc.want {
				t.Fatalf("strategy = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestUnsafeStrategiesDoNotExposeAutomaticDelete(t *testing.T) {
	for _, s := range []catalogpkg.UninstallStrategy{catalogpkg.StrategySystemComponentUnsupported, catalogpkg.StrategyManualOnly, catalogpkg.StrategyUnresolved} {
		if strategyAllowsAutomaticUninstall(s) {
			t.Fatalf("unsafe strategy exposed automatic uninstall: %s", s)
		}
	}
}
