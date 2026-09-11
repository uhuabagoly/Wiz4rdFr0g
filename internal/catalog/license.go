package catalog

import "strings"

type LicenseClass string

const (
	LicenseOpenSource      LicenseClass = "open_source"
	LicenseFreeware        LicenseClass = "freeware"
	LicenseFreeTier        LicenseClass = "free_tier"
	LicenseTrial           LicenseClass = "trial"
	LicenseCommercial      LicenseClass = "commercial"
	LicenseSystemComponent LicenseClass = "system_component"
	LicenseUnknown         LicenseClass = "unknown"
)

type LicenseMetadata struct {
	Class     LicenseClass `json:"class"`
	SourceURL string       `json:"source_url,omitempty"`
	CheckedAt string       `json:"checked_at,omitempty"`
	Note      string       `json:"note,omitempty"`
	PolicyOK  bool         `json:"policy_ok"`
}

// licenseOverrides contains only classifications backed by an explicit source.
// Every catalog item not listed here remains unknown and therefore release-blocking.
// Do not infer license class from package availability, Winget presence or product name.
var licenseOverrides = map[string]LicenseMetadata{
	"7-Zip": {
		Class: LicenseOpenSource, SourceURL: "https://www.7-zip.org/", CheckedAt: "2026-09-11",
		Note: "Official 7-Zip site states that 7-Zip is free software/open source and may be used without registration or payment.",
	},
	"Blender": {
		Class: LicenseOpenSource, SourceURL: "https://www.blender.org/about/license/", CheckedAt: "2026-09-11",
		Note: "Official Blender license page: GNU GPL free software, usable for any purpose.",
	},
	"Krita": {
		Class: LicenseOpenSource, SourceURL: "https://krita.org/en/about/license/", CheckedAt: "2026-09-11",
		Note: "Official Krita license page: GNU GPL v3, free to use for any purpose.",
	},
	"Microsoft Word": {
		Class: LicenseCommercial, SourceURL: "https://www.microsoft.com/hu-hu/microsoft-365/buy/compare-all-microsoft-365-products", CheckedAt: "2026-09-11",
		Note: "Desktop Word is included in paid Microsoft 365/Office offerings; it does not satisfy the free-desktop-app catalog policy.",
	},
	"Microsoft Excel": {
		Class: LicenseCommercial, SourceURL: "https://www.microsoft.com/hu-hu/microsoft-365/buy/compare-all-microsoft-365-products", CheckedAt: "2026-09-11",
		Note: "Desktop Excel is included in paid Microsoft 365/Office offerings; it does not satisfy the free-desktop-app catalog policy.",
	},
	"Microsoft PowerPoint": {
		Class: LicenseCommercial, SourceURL: "https://www.microsoft.com/hu-hu/microsoft-365/buy/compare-all-microsoft-365-products", CheckedAt: "2026-09-11",
		Note: "Desktop PowerPoint is included in paid Microsoft 365/Office offerings; free web access is not equivalent to the installable desktop product.",
	},
	"Microsoft Access": {
		Class: LicenseCommercial, SourceURL: "https://www.microsoft.com/hu-hu/microsoft-365/buy/compare-all-microsoft-365-products", CheckedAt: "2026-09-11",
		Note: "Microsoft Access desktop licensing is commercial and does not satisfy the free-app catalog policy.",
	},
}

func LicenseFor(app AppDef, systemComponent bool) LicenseMetadata {
	if systemComponent {
		return LicenseMetadata{Class: LicenseSystemComponent, Note: "Windows-managed system component; excluded from the normal application license policy."}
	}
	m, ok := licenseOverrides[app.Name]
	if !ok {
		return LicenseMetadata{Class: LicenseUnknown, Note: "License has not yet been verified against an official source."}
	}
	m.PolicyOK = LicensePolicyAllows(m)
	return m
}

func LicensePolicyAllows(m LicenseMetadata) bool {
	if strings.TrimSpace(m.SourceURL) == "" || strings.TrimSpace(m.CheckedAt) == "" {
		return false
	}
	switch m.Class {
	case LicenseOpenSource, LicenseFreeware:
		return true
	default:
		return false
	}
}

func LicenseRequiresEvidence(class LicenseClass) bool {
	switch class {
	case LicenseUnknown, LicenseSystemComponent:
		return false
	default:
		return true
	}
}
