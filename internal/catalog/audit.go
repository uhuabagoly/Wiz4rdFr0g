package catalog

import (
	"fmt"
	"sort"
	"strings"
)

type AuditEntry struct {
	Index                      int               `json:"index"`
	Name                       string            `json:"name"`
	Category                   string            `json:"category"`
	CanonicalName              string            `json:"canonical_name"`
	CatalogAliasOf             string            `json:"catalog_alias_of,omitempty"`
	CatalogValid               bool              `json:"catalog_valid"`
	PlatformValid              bool              `json:"platform_valid"`
	Platform                   Platform          `json:"platform"`
	WingetID                   string            `json:"winget_id,omitempty"`
	WingetIDKnown              bool              `json:"winget_id_known"`
	WingetIDVerified           bool              `json:"winget_id_verified"`
	WingetManifestURL          string            `json:"winget_manifest_url,omitempty"`
	AlternativeIDs             []string          `json:"alternative_ids,omitempty"`
	DisplayAliases             []string          `json:"display_aliases"`
	RegistryMatchMode          string            `json:"registry_match_mode"`
	ForbiddenRegistryVariants  []string          `json:"forbidden_registry_variants"`
	PackageResolutionSafe      bool              `json:"package_resolution_safe"`
	ResolutionStatus           ResolutionStatus  `json:"resolution_status"`
	ResolutionEvidence         string            `json:"resolution_evidence"`
	DetectionStrategy          string            `json:"detection_strategy"`
	ExpectedScope              ExpectedScope     `json:"expected_scope"`
	InstallerType              InstallerType     `json:"installer_type"`
	UninstallStrategy          UninstallStrategy `json:"uninstall_strategy"`
	UninstallSupport           UninstallSupport  `json:"uninstall_support"`
	SystemComponent            bool              `json:"system_component"`
	PhysicalTestRequired       bool              `json:"physical_test_required"`
	DefaultLocationInformative bool              `json:"default_install_location_informational"`
	UnresolvedReason           string            `json:"unresolved_reason,omitempty"`
}

type AuditSummary struct {
	CatalogEntries               int            `json:"catalog_entries"`
	CanonicalPrograms            int            `json:"canonical_programs"`
	CatalogAliases               int            `json:"catalog_aliases"`
	UniqueNames                  int            `json:"unique_names"`
	DuplicateNames               int            `json:"duplicate_names"`
	HardcodedIDEntries           int            `json:"hardcoded_id_entries"`
	UniqueHardcodedIDs           int            `json:"unique_hardcoded_ids"`
	VerifiedHardcodedIDEntries   int            `json:"verified_hardcoded_id_entries"`
	VerifiedUniqueHardcodedIDs   int            `json:"verified_unique_hardcoded_ids"`
	UnverifiedHardcodedIDEntries int            `json:"unverified_hardcoded_id_entries"`
	ExactNameVerifiedEntries     int            `json:"exact_name_verified_entries"`
	RuntimeExactRequiredEntries  int            `json:"runtime_exact_required_entries"`
	SystemComponents             int            `json:"system_components"`
	MSIXStore                    int            `json:"msix_store"`
	ManualOnly                   int            `json:"manual_only"`
	Unresolved                   int            `json:"unresolved"`
	PhysicalTestRequired         int            `json:"physical_test_required"`
	UnsafeResolutionEntries      int            `json:"unsafe_resolution_entries"`
	StrategyCounts               map[string]int `json:"strategy_counts"`
	ResolutionCounts             map[string]int `json:"resolution_counts"`
	ValidationErrors             []string       `json:"validation_errors"`
	Pass                         bool           `json:"pass"`
}

type AuditReport struct {
	GeneratedAt string       `json:"generated_at"`
	Scope       string       `json:"scope"`
	Summary     AuditSummary `json:"summary"`
	Entries     []AuditEntry `json:"entries"`
}

func BuildAuditEntries() []AuditEntry {
	out := make([]AuditEntry, 0, len(Entries))
	for i, app := range Entries {
		p := ProfileFor(app)
		e := AuditEntry{
			Index:                      i,
			Name:                       app.Name,
			Category:                   app.Category,
			CanonicalName:              p.CanonicalName,
			CatalogAliasOf:             p.CatalogAliasOf,
			CatalogValid:               strings.TrimSpace(app.Name) != "" && strings.TrimSpace(app.Category) != "",
			PlatformValid:              p.Platform == PlatformWindows,
			Platform:                   p.Platform,
			WingetID:                   p.WingetID,
			WingetIDKnown:              strings.TrimSpace(p.WingetID) != "",
			WingetIDVerified:           p.ResolutionStatus == ResolutionHardcodedVerified,
			WingetManifestURL:          wingetManifestURL(p.WingetID),
			AlternativeIDs:             append([]string(nil), p.AlternativeIDs...),
			DisplayAliases:             append([]string(nil), p.DisplayAliases...),
			RegistryMatchMode:          p.RegistryMatchMode,
			ForbiddenRegistryVariants:  append([]string(nil), p.ForbiddenRegistryVariants...),
			PackageResolutionSafe:      p.PackageResolutionSafe,
			ResolutionStatus:           p.ResolutionStatus,
			ResolutionEvidence:         p.ResolutionEvidence,
			DetectionStrategy:          "exact-package-id -> exact-display-alias -> strict-registry-alias",
			ExpectedScope:              p.ExpectedScope,
			InstallerType:              p.InstallerType,
			UninstallStrategy:          p.UninstallStrategy,
			UninstallSupport:           p.UninstallSupport,
			SystemComponent:            p.SystemComponent,
			PhysicalTestRequired:       p.PhysicalTestRequired,
			DefaultLocationInformative: p.DefaultInstallLocationInformational,
		}
		if p.ResolutionStatus == ResolutionHardcodedUnverified {
			e.UnresolvedReason = "hardcoded package ID requires exact runtime revalidation"
		}
		if p.ResolutionStatus == ResolutionRuntimeExact {
			e.UnresolvedReason = "no static package ID; exact runtime source resolution is required before install or uninstall"
		}
		out = append(out, e)
	}
	return out
}

func wingetManifestURL(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	parts := strings.Split(id, ".")
	if len(parts) < 2 || parts[0] == "" {
		return ""
	}
	first := strings.ToLower(parts[0][:1])
	return "https://github.com/microsoft/winget-pkgs/tree/master/manifests/" + first + "/" + strings.Join(parts, "/")
}

func Summarize(entries []AuditEntry) AuditSummary {
	s := AuditSummary{StrategyCounts: map[string]int{}, ResolutionCounts: map[string]int{}}
	s.CatalogEntries = len(entries)
	nameCounts := map[string]int{}
	idCounts := map[string]int{}
	verifiedIDs := map[string]bool{}
	canonicalPrimaries := map[string]bool{}
	for _, e := range entries {
		nameCounts[strings.ToLower(strings.TrimSpace(e.Name))]++
		if e.CatalogAliasOf == "" {
			canonicalPrimaries[strings.ToLower(strings.TrimSpace(e.CanonicalName))] = true
		} else {
			s.CatalogAliases++
		}
		if e.WingetIDKnown {
			s.HardcodedIDEntries++
			idCounts[strings.ToLower(e.WingetID)]++
		}
		if e.WingetIDVerified {
			s.VerifiedHardcodedIDEntries++
			verifiedIDs[strings.ToLower(e.WingetID)] = true
		}
		if e.ResolutionStatus == ResolutionHardcodedUnverified {
			s.UnverifiedHardcodedIDEntries++
		}
		if e.ResolutionStatus == ResolutionExactNameVerified {
			s.ExactNameVerifiedEntries++
		}
		if e.ResolutionStatus == ResolutionRuntimeExact {
			s.RuntimeExactRequiredEntries++
		}
		if e.SystemComponent {
			s.SystemComponents++
		}
		if e.UninstallStrategy == StrategyMSIXStore {
			s.MSIXStore++
		}
		if e.UninstallStrategy == StrategyManualOnly {
			s.ManualOnly++
		}
		if e.UninstallStrategy == StrategyUnresolved || e.ResolutionStatus == ResolutionHardcodedUnverified || e.ResolutionStatus == ResolutionRuntimeExact {
			s.Unresolved++
		}
		if e.PhysicalTestRequired {
			s.PhysicalTestRequired++
		}
		if !e.PackageResolutionSafe {
			s.UnsafeResolutionEntries++
		}
		s.StrategyCounts[string(e.UninstallStrategy)]++
		s.ResolutionCounts[string(e.ResolutionStatus)]++
	}
	for _, n := range nameCounts {
		if n == 1 {
			s.UniqueNames++
		} else {
			s.DuplicateNames += n - 1
		}
	}
	s.CanonicalPrograms = len(canonicalPrimaries)
	s.UniqueHardcodedIDs = len(idCounts)
	s.VerifiedUniqueHardcodedIDs = len(verifiedIDs)
	s.ValidationErrors = ValidateCatalog(entries)
	s.Pass = len(s.ValidationErrors) == 0 && s.UnsafeResolutionEntries == 0
	return s
}

func ValidateCatalog(entries []AuditEntry) []string {
	var errs []string
	if len(entries) == 0 {
		return []string{"catalog is empty"}
	}
	nameSeen := map[string]int{}
	primaryCanonical := map[string]string{}
	idOwners := map[string][]AuditEntry{}
	for _, e := range entries {
		if !e.CatalogValid {
			errs = append(errs, fmt.Sprintf("invalid catalog entry at index %d", e.Index))
		}
		if !e.PlatformValid {
			errs = append(errs, fmt.Sprintf("invalid platform for %s", e.Name))
		}
		if len(e.DisplayAliases) == 0 {
			errs = append(errs, fmt.Sprintf("missing display aliases for %s", e.Name))
		}
		if strings.TrimSpace(e.RegistryMatchMode) == "" {
			errs = append(errs, fmt.Sprintf("missing registry match mode for %s", e.Name))
		}
		if len(e.ForbiddenRegistryVariants) == 0 {
			errs = append(errs, fmt.Sprintf("missing forbidden registry variants for %s", e.Name))
		}
		if !e.PackageResolutionSafe {
			errs = append(errs, fmt.Sprintf("unsafe package resolution for %s", e.Name))
		}
		key := strings.ToLower(strings.TrimSpace(e.Name))
		nameSeen[key]++
		if e.CatalogAliasOf == "" {
			ck := strings.ToLower(strings.TrimSpace(e.CanonicalName))
			if other, ok := primaryCanonical[ck]; ok {
				errs = append(errs, fmt.Sprintf("duplicate canonical program: %s and %s", other, e.Name))
			} else {
				primaryCanonical[ck] = e.Name
			}
		}
		if e.WingetIDKnown {
			idOwners[strings.ToLower(e.WingetID)] = append(idOwners[strings.ToLower(e.WingetID)], e)
		}
	}
	for n, c := range nameSeen {
		if c > 1 {
			errs = append(errs, fmt.Sprintf("duplicate catalog name: %s", n))
		}
	}
	for id, owners := range idOwners {
		if len(owners) <= 1 {
			continue
		}
		primary := 0
		for _, o := range owners {
			if o.CatalogAliasOf == "" {
				primary++
			}
		}
		if primary > 1 {
			names := make([]string, 0, len(owners))
			for _, o := range owners {
				names = append(names, o.Name)
			}
			sort.Strings(names)
			errs = append(errs, fmt.Sprintf("package ID %s is assigned to multiple primary programs: %s", id, strings.Join(names, ", ")))
		}
	}
	sort.Strings(errs)
	return errs
}
