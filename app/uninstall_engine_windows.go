//go:build windows

package main

import (
	"context"
	"strings"

	catalogpkg "wiz4rdfr0g.local/fullcatalog/internal/catalog"
)

func executeInstalledUninstall(ctx context.Context, app appDef, pkg installedPackage, registryPackages []registryPackage, elevated bool) int {
	scope := strings.ToLower(strings.TrimSpace(pkg.Scope))
	if isMicrosoftEdgeApp(app) {
		if !strings.EqualFold(strings.TrimSpace(pkg.ID), "Microsoft.Edge") {
			return uninstallCodeUnsupported
		}
		if scope == "user" && elevated {
			return uninstallCodeWrongElevation
		}
		if scope == "machine" && !elevated {
			return uninstallCodeNeedElevation
		}
		return uninstallAttemptCode(app, runWingetUninstallScoped(ctx, app, pkg, true, scope))
	}
	strategy := installedUninstallStrategy(app, pkg)
	if scope == "user" && elevated {
		return uninstallCodeWrongElevation
	}
	if scope == "machine" && !elevated {
		return uninstallCodeNeedElevation
	}
	switch strategy {
	case catalogpkg.StrategySystemComponentUnsupported, catalogpkg.StrategyManualOnly, catalogpkg.StrategyUnresolved:
		return uninstallCodeUnsupported
	case catalogpkg.StrategyMSIXStore:
		attempt := runWingetUninstallScoped(ctx, app, pkg, true, scope)
		return uninstallAttemptCode(app, attempt)
	case catalogpkg.StrategyMSIProductCode:
		if reg, ok := resolveRegistryForInstalled(app, pkg, registryPackages); ok {
			attempt := runRegisteredUninstaller(ctx, app, reg)
			code := uninstallAttemptCode(app, attempt)
			if code == uninstallCodeOK || code == uninstallCodeRebootRequired || code == uninstallCodeRunningProcess || code == uninstallCodeNeedElevation || code == uninstallCodeWrongElevation {
				return code
			}
		}
		if strings.TrimSpace(pkg.ID) != "" {
			return uninstallAttemptCode(app, runWingetUninstallScoped(ctx, app, pkg, true, scope))
		}
		return uninstallCodeFailed
	case catalogpkg.StrategyRegistryUser, catalogpkg.StrategyRegistryMachine, catalogpkg.StrategyVendorUninstaller:
		if reg, ok := resolveRegistryForInstalled(app, pkg, registryPackages); ok {
			attempt := runRegisteredUninstaller(ctx, app, reg)
			code := uninstallAttemptCode(app, attempt)
			if code == uninstallCodeOK || code == uninstallCodeRebootRequired || code == uninstallCodeRunningProcess || code == uninstallCodeNeedElevation || code == uninstallCodeWrongElevation {
				return code
			}
		}
		if strings.TrimSpace(pkg.ID) != "" {
			return uninstallAttemptCode(app, runWingetUninstallScoped(ctx, app, pkg, true, scope))
		}
		return uninstallCodeFailed
	case catalogpkg.StrategyWingetUser, catalogpkg.StrategyWingetMachine, catalogpkg.StrategyRuntimeDetect:
		attempt := runWingetUninstallScoped(ctx, app, pkg, true, scope)
		code := uninstallAttemptCode(app, attempt)
		if code == uninstallCodeOK || code == uninstallCodeRebootRequired || code == uninstallCodeRunningProcess || code == uninstallCodeNeedElevation || code == uninstallCodeWrongElevation {
			return code
		}
		if reg, ok := resolveRegistryForInstalled(app, pkg, registryPackages); ok {
			return uninstallAttemptCode(app, runRegisteredUninstaller(ctx, app, reg))
		}
		return uninstallCodeFailed
	default:
		return uninstallCodeUnsupported
	}
}

func uninstallAttemptCode(app appDef, attempt uninstallAttempt) int {
	code := classifyUninstallAttempt(attempt)
	if code != uninstallCodeOK {
		return code
	}
	if verifyProgramRemoved(app) {
		return uninstallCodeOK
	}
	return uninstallCodeFailed
}
