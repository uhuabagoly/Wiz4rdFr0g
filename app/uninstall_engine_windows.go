//go:build windows

package main

import (
	"context"
	"fmt"
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
	workerLog("INFO", fmt.Sprintf("%s: uninstall target id=%q name=%q scope=%q strategy=%q registry=%q", app.Name, pkg.ID, pkg.Name, scope, strategy, pkg.RegistryKey))
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
			workerLog("REPAIR", app.Name+": registered/MSI uninstall path failed; retrying the same exact target through Winget ID.")
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
			workerLog("REPAIR", app.Name+": registered vendor uninstall path failed; retrying the same exact target through Winget ID.")
			return uninstallAttemptCode(app, runWingetUninstallScoped(ctx, app, pkg, true, scope))
		}
		return uninstallCodeFailed
	case catalogpkg.StrategyWingetUser, catalogpkg.StrategyWingetMachine, catalogpkg.StrategyRuntimeDetect:
		// User-scoped Winget uninstall can stall before invoking the vendor
		// uninstaller. Prefer the exact registration's declared quiet command
		// when available, while keeping the same standard-user context.
		if scope == "user" {
			if reg, ok := resolveRegistryForInstalled(app, pkg, registryPackages); ok && strings.TrimSpace(reg.QuietUninstallString) != "" {
				workerLog("INFO", app.Name+": using the matched vendor QuietUninstallString in the owning user's context.")
				code := uninstallAttemptCode(app, runRegisteredUninstaller(ctx, app, reg))
				if code != uninstallCodeFailed && code != uninstallCodeUnsupported {
					return code
				}
			}
		}
		attempt := runWingetUninstallScoped(ctx, app, pkg, true, scope)
		code := uninstallAttemptCode(app, attempt)
		if code == uninstallCodeOK || code == uninstallCodeRebootRequired || code == uninstallCodeRunningProcess || code == uninstallCodeNeedElevation || code == uninstallCodeWrongElevation {
			return code
		}
		if reg, ok := resolveRegistryForInstalled(app, pkg, registryPackages); ok {
			workerLog("REPAIR", app.Name+": exact Winget uninstall failed; retrying the matched registered uninstaller for the same detected application.")
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
	workerLog("WARN", app.Name+": uninstall command reported success but independent detector still finds the target.")
	return uninstallCodeFailed
}
