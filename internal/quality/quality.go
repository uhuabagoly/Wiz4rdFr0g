package quality

import "strings"

type RootCause string

type CoverageStatus string

const (
	RootNone                    RootCause = "None"
	RootWrongPackageID          RootCause = "WrongPackageId"
	RootAmbiguousMatch          RootCause = "AmbiguousMatch"
	RootWrongRegistryMatch      RootCause = "WrongRegistryMatch"
	RootWrongScope              RootCause = "WrongScope"
	RootUserScopeElevated       RootCause = "UserScopeElevated"
	RootMachineScopeNotElevated RootCause = "MachineScopeNotElevated"
	RootBrokenUninstallString   RootCause = "BrokenUninstallString"
	RootBrokenQuotedPath        RootCause = "BrokenQuotedPath"
	RootMSIProductCode          RootCause = "MSIProductCode"
	RootMSIXStore               RootCause = "MSIXStore"
	RootRebootRequired          RootCause = "RebootRequired"
	RootRunningProcess          RootCause = "RunningProcess"
	RootSystemComponent         RootCause = "SystemComponent"
	RootUnsupportedAutomation   RootCause = "UnsupportedAutomation"
	RootInstallerChanged        RootCause = "InstallerChanged"
	RootPackageUnavailable      RootCause = "PackageUnavailable"
	RootDetectionFalsePositive  RootCause = "DetectionFalsePositive"
	RootDetectionFalseNegative  RootCause = "DetectionFalseNegative"
	RootLicensePolicy           RootCause = "LicensePolicyBlocked"
	RootTimeout                 RootCause = "Timeout"
	RootTransientPackageManager RootCause = "TransientPackageManager"
	RootUninstallRepairFailed   RootCause = "UninstallRepairFailed"
	RootOther                   RootCause = "Other"

	CoverageVerifiedFull        CoverageStatus = "VERIFIED_FULL"
	CoverageVerifiedInstallOnly CoverageStatus = "VERIFIED_INSTALL_ONLY"
	CoverageManualUninstall     CoverageStatus = "MANUAL_UNINSTALL"
	CoverageSystemComponent     CoverageStatus = "SYSTEM_COMPONENT"
	CoverageLicenseBlocked      CoverageStatus = "LICENSE_BLOCKED"
	CoverageUnavailable         CoverageStatus = "UNAVAILABLE"
	CoverageUnresolved          CoverageStatus = "UNRESOLVED"
)

type ResultFacts struct {
	FinalStatus       string
	FailureStage      string
	Failure           string
	SkipReason        string
	InstallVerified   bool
	UninstallVerified bool
}

func ClassifyRootCause(f ResultFacts) RootCause {
	status := strings.ToUpper(strings.TrimSpace(f.FinalStatus))
	stage := strings.ToUpper(strings.TrimSpace(f.FailureStage))
	text := strings.ToLower(strings.Join([]string{f.Failure, f.SkipReason}, " "))
	if status == "FULL_PASS" {
		return RootNone
	}
	if status == "SYSTEM_COMPONENT" || strings.Contains(text, "system component") || strings.Contains(text, "rendszerkomponens") {
		return RootSystemComponent
	}
	if status == "LICENSE_REQUIRED" || strings.Contains(text, "license policy") || strings.Contains(text, "licence policy") {
		return RootLicensePolicy
	}
	if strings.Contains(status, "TIMEOUT") || strings.Contains(text, "context deadline exceeded") || strings.Contains(text, "timed out") || strings.Contains(text, "timeout") {
		return RootTimeout
	}
	if strings.Contains(text, "transient package") || strings.Contains(text, "temporary source") || strings.Contains(text, "network retry") {
		return RootTransientPackageManager
	}
	if status == "UNINSTALL_REPAIR_FAILED" || strings.Contains(text, "repair attempts exhausted") {
		return RootUninstallRepairFailed
	}
	if status == "SKIPPED_REBOOT_REQUIRED" || strings.Contains(text, "reboot") || strings.Contains(text, "restart required") || strings.Contains(text, "újraindítás") {
		return RootRebootRequired
	}
	if strings.Contains(text, "installed for user scope cannot be uninstalled") || strings.Contains(text, "user-scope uninstall must run with a standard user token") {
		return RootUserScopeElevated
	}
	if strings.Contains(text, "machine-scope uninstall requires") || strings.Contains(text, "requires elevation") || strings.Contains(text, "administrator privileges are required") {
		return RootMachineScopeNotElevated
	}
	if strings.Contains(text, "running") || strings.Contains(text, "in use") || strings.Contains(text, "close the application") || strings.Contains(text, "close app") || strings.Contains(text, "program is open") {
		return RootRunningProcess
	}
	if strings.Contains(text, "ambiguous") || strings.Contains(text, "multiple exact") || strings.Contains(text, "több pontos") {
		return RootAmbiguousMatch
	}
	if strings.Contains(text, "wrong package") || strings.Contains(text, "package id mismatch") || strings.Contains(text, "resolved id mismatch") {
		return RootWrongPackageID
	}
	if strings.Contains(text, "registry") && (strings.Contains(text, "wrong match") || strings.Contains(text, "mismatch") || strings.Contains(text, "incorrect match")) {
		return RootWrongRegistryMatch
	}
	if strings.Contains(text, "scope") && (strings.Contains(text, "wrong") || strings.Contains(text, "mismatch") || strings.Contains(text, "incorrect")) {
		return RootWrongScope
	}
	if strings.Contains(text, "uninstallstring") || strings.Contains(text, "uninstall string") || strings.Contains(text, "registry uninstall") && strings.Contains(text, "parse") {
		return RootBrokenUninstallString
	}
	if strings.Contains(text, "quoted path") || strings.Contains(text, "idéző") || strings.Contains(text, "unquoted path") {
		return RootBrokenQuotedPath
	}
	if strings.Contains(text, "productcode") || strings.Contains(text, "product code") || strings.Contains(text, "msiexec") || strings.Contains(text, "msi") && stage == "UNINSTALL" {
		return RootMSIProductCode
	}
	if strings.Contains(text, "msstore") || strings.Contains(text, "microsoft store") || strings.Contains(text, "msix") || strings.Contains(status, "STORE") {
		return RootMSIXStore
	}
	if strings.Contains(text, "installer changed") || strings.Contains(text, "hash mismatch") || strings.Contains(text, "installer hash") {
		return RootInstallerChanged
	}
	if status == "UNAVAILABLE" || strings.Contains(text, "unavailable") || strings.Contains(text, "no exact production package resolution") || strings.Contains(text, "package not found") || strings.Contains(text, "download unavailable") {
		return RootPackageUnavailable
	}
	if status == "MANUAL_ONLY" || strings.Contains(text, "manual uninstall") || strings.Contains(text, "no safe automatic uninstall strategy") {
		return RootUnsupportedAutomation
	}
	if stage == "DETECTION" || stage == "INSTALL_VERIFY" || stage == "DETECTION_AFTER_REBOOT" || stage == "INSTALL_VERIFY_AFTER_REBOOT" {
		return RootDetectionFalseNegative
	}
	if stage == "PRECHECK" && strings.Contains(text, "already existed") {
		return RootDetectionFalsePositive
	}
	return RootOther
}

func CoverageFor(f ResultFacts) CoverageStatus {
	status := strings.ToUpper(strings.TrimSpace(f.FinalStatus))
	switch status {
	case "FULL_PASS":
		if f.InstallVerified && f.UninstallVerified {
			return CoverageVerifiedFull
		}
	case "MANUAL_ONLY":
		return CoverageManualUninstall
	case "SYSTEM_COMPONENT":
		return CoverageSystemComponent
	case "LICENSE_REQUIRED":
		return CoverageLicenseBlocked
	case "UNAVAILABLE", "SKIPPED_STORE_ENVIRONMENT":
		return CoverageUnavailable
	}
	if f.InstallVerified && !f.UninstallVerified {
		return CoverageVerifiedInstallOnly
	}
	return CoverageUnresolved
}
