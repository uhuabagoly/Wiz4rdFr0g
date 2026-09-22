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
	"Mozilla Firefox":     {Class: LicenseOpenSource, SourceURL: "https://www.mozilla.org/en-US/MPL/2.0/", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for Mozilla.Firefox; source SHA256 79fefbb5ce750421056e8639e83b055fa97deb02407617d4c0f9d50cfa0cfd32. Physical lifecycle still required."},
	"Mozilla Thunderbird": {Class: LicenseOpenSource, SourceURL: "https://www.mozilla.org/en-US/MPL/2.0/", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for Mozilla.Thunderbird; source SHA256 79fefbb5ce750421056e8639e83b055fa97deb02407617d4c0f9d50cfa0cfd32. Physical lifecycle still required."},
	"LibreOffice":         {Class: LicenseOpenSource, SourceURL: "https://www.libreoffice.org/licenses/", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for TheDocumentFoundation.LibreOffice; source SHA256 fc8c0e5c724cf292e03af59d3241fccc8b29f1120f54c731a46240fcea62ee3b. Physical lifecycle still required."},
	"PeaZip":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/peazip/PeaZip/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official LGPL-3.0 terms for Giorgiotani.Peazip; source SHA256 a853c2ffec17057872340eee242ae4d96cbf2b520ae27d903e1b2fef1a5f9d1c. Physical lifecycle still required."},
	"Python 3":            {Class: LicenseOpenSource, SourceURL: "https://docs.python.org/3/license.html", CheckedAt: "2026-09-22", Note: "Reviewed official PSF-2.0 terms for Python.Python.3.13; source SHA256 287275506de7ece16530519cd9474b1bc5c63eb8b6010d0f713359ec18931a8a. Physical lifecycle still required."},
	"PuTTY":               {Class: LicenseOpenSource, SourceURL: "https://www.chiark.greenend.org.uk/~sgtatham/putty/licence.html", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for PuTTY.PuTTY; source SHA256 643a9142655b672ef53f729dded917cff6e0f9dfa22d32629ff5ec137e15e3c5. Physical lifecycle still required."},
	"VirtualBox":          {Class: LicenseOpenSource, SourceURL: "https://www.virtualbox.org/wiki/Licensing_FAQ", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0-only terms for Oracle.VirtualBox; source SHA256 02563e1fcebacf1917e17e9a2d51dd33365c05cfb6f78099e9fb4820aa39e3ff. Physical lifecycle still required. Covers base package only, not the Extension Pack."},
	"Playnite":            {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/JosefNemec/Playnite/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for Playnite.Playnite; source SHA256 10c0871ee7dd3ac9f6ef53097a173113229edd1b23e9218d952287889b2b84eb. Physical lifecycle still required."},
	"Python":              {Class: LicenseOpenSource, SourceURL: "https://docs.python.org/3/license.html", CheckedAt: "2026-09-22", Note: "Reviewed official PSF-2.0 terms for Python.Python.3.13; source SHA256 287275506de7ece16530519cd9474b1bc5c63eb8b6010d0f713359ec18931a8a. Physical lifecycle still required."},
	"FFmpeg":              {Class: LicenseOpenSource, SourceURL: "https://www.ffmpeg.org/legal.html", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Gyan.FFmpeg; source SHA256 cd51dc25f9e3200b1e3953aee036f499ed51f01f7ba30563873186cb3f870eda. Physical lifecycle still required."},
	"KeePass":             {Class: LicenseOpenSource, SourceURL: "https://keepass.info/help/v2/license.html", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for DominikReichl.KeePass; source SHA256 c5f8a4e39f723cb1ead3175f0f5b33a51feabf56ad8e5977562f792ace061995. Physical lifecycle still required."},
	"KeePassXC":           {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/keepassxreboot/keepassxc/HEAD/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0-or-later terms for KeePassXCTeam.KeePassXC; source SHA256 f4cf558763d725e47a55da6f32735b3f0b3d69184870f921015b3c54b58bfb36. Physical lifecycle still required."},
	"Brave":               {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/brave/brave-browser/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MPL-2.0 terms for Brave.Brave; source SHA256 3f3d9e0024b1921b067d6f7f88deb4a60cbe7a78e76c64e3f1d7fc3b779b9d04. Source collection CI 35685779903; physical lifecycle still required."},
	"SumatraPDF":          {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/sumatrapdfreader/sumatrapdf/3.6rel/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for SumatraPDF.SumatraPDF; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. Source collection CI 35685779903; physical lifecycle still required."},
	"Everything":          {Class: LicenseOpenSource, SourceURL: "https://www.voidtools.com/License.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for voidtools.Everything; source SHA256 c13d19adcbfd5d07e9512de9df99956a3423399ed1fadc5fd33186697ad8df2f. Source collection CI 35685779903; physical lifecycle still required."},
	"WinDirStat":          {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/windirstat/windirstat/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-2.0 terms for WinDirStat.WinDirStat; source SHA256 5d05a329bdd65bc1212fda57942ea90a2a810ff7ce192dc38d7042a55caa4a28. Source collection CI 35685779903; physical lifecycle still required."},
	"Rufus":               {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/pbatard/rufus/HEAD/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0-or-later terms for Rufus.Rufus; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. Source collection CI 35685779903; physical lifecycle still required."},
	"balenaEtcher":        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/balena-io/etcher/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project Apache-2.0 terms for Balena.Etcher; source SHA256 39445b459f86621683f9731fb6a7d070819dc379840e8ef52f62bb1c68942291. Source collection CI 35685779903; physical lifecycle still required."},
	"Windows Terminal":    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/microsoft/terminal/main/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for Microsoft.WindowsTerminal; source SHA256 5d177f23ecfeb0ea8e050b6a5a16355e1ae9a0b286436ca8f83ed08b3795be6b. Source collection CI 35685779903; physical lifecycle still required."},
	"WinSCP":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/winscp/winscp/HEAD/license.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0-or-later terms for WinSCP.WinSCP; source SHA256 38493e2a77946bb116831c1a394b5f9d073b8490be20ebc6b5e42df64237dbaa. Source collection CI 35685779903; physical lifecycle still required."},
	"DBeaver":             {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/dbeaver/dbeaver/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official project Apache-2.0 terms for DBeaver.DBeaver.Community; source SHA256 3574a35d6ceb85fe03ed8dee865ebfcfbbd7c6de6899e890966f437ffe177373. Source collection CI 35685779903; physical lifecycle still required."},
	"Godot Engine":        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/godotengine/godot/HEAD/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for GodotEngine.GodotEngine; source SHA256 b0435e3b3e4e55238f05f4b306f30524a1b2e20147810d436eaa554fa6855c80. Source collection CI 35685779903; physical lifecycle still required."},
	"CMake":               {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/Kitware/CMake/HEAD/LICENSE.rst", CheckedAt: "2026-09-22", Note: "Reviewed official project BSD-3-Clause terms for Kitware.CMake; source SHA256 4382e7c1879ac90e3f101a395d23846fa4dbcaa1eed7265b43681e348754825d. Source collection CI 35685779903; physical lifecycle still required."},
	"Ventoy":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ventoy/Ventoy/master/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for Ventoy.Ventoy; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. Source collection CI 35685779903; physical lifecycle still required."},
	"Ollama":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ollama/ollama/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for Ollama.Ollama; source SHA256 5934ed2ce0d15154bcdb9c85203210abac0da4314af34081e36df4599f90b226. Source collection CI 35685779903; physical lifecycle still required."},
	"GitHub CLI":          {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/cli/cli/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for GitHub.cli; source SHA256 6da4adc42392c8485e40b4251c7e332fc3352df1947c9ffade71dd60b14a7a4f. Source collection CI 35685779903; physical lifecycle still required."},
	"Shotcut":             {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/mltframework/shotcut/HEAD/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for Meltytech.Shotcut; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. Source collection CI 35685779903; physical lifecycle still required."},
	"OpenShot":            {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/OpenShot/openshot-qt/HEAD/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for OpenShot.OpenShot; source SHA256 667da052f7e5e0c1e49b6921a1811b536ff6e8cfb9719e628dc08dae487925d2. Source collection CI 35685779903; physical lifecycle still required."},
	"GIMP":                {Class: LicenseOpenSource, SourceURL: "https://www.gimp.org/about/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. No physical PASS implied."},
	"HandBrake":           {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/HandBrake/HandBrake/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. No physical PASS implied."},
	"OBS Studio":          {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/obsproject/obs-studio/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. No physical PASS implied."},
	"ShareX":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ShareX/ShareX/master/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 c53a65c2fd561c87eaabf1072ef5dcab8653042bc15308465f52413585eb6271. No physical PASS implied."},
	"Greenshot":           {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/greenshot/greenshot/develop/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. No physical PASS implied."},
	"qBittorrent":         {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/qbittorrent/qBittorrent/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 e675cd856f9817474455200ba7e6f5b7cc42d6598a5eecbbbdaa0e6fd304d6b7. No physical PASS implied."},
	"Telegram Desktop":    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/telegramdesktop/tdesktop/dev/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 a728b65d02eeaf6b50c075a33b76107c652c3e5a12ff67f53682c83d47a23e1e. No physical PASS implied."},
	"Git":                 {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/git/git/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 5b2198d1645f767585e8a88ac0499b04472164c0d2da22e75ecf97ef443ab32e. No physical PASS implied."},
	"Microsoft PowerToys": {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/microsoft/PowerToys/main/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 1eb3cbe7f022addbfc6d65cb2e39aca5b24333942ed843aa10c03befd71ee225. No physical PASS implied."},
	"Notepad++":           {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/notepad-plus-plus/notepad-plus-plus/master/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 2b94f58d89424af06d1a8e16775774757f1ecfb678203c3439af037a24f35dc6. No physical PASS implied."},
	"GitHub Desktop":      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/desktop/desktop/development/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 891d678cd6aa67c0712f663b5fee690f24d11d360795300814f7bf2eb91ba530. No physical PASS implied."},
	"Node.js":             {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/nodejs/node/main/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 37110192cd7621a80510e2f2630ae08f0420f97257d4ae1c4d51c11261f1f4b7. No physical PASS implied."},
	"Audacity": {
		Class: LicenseOpenSource, SourceURL: "https://www.audacityteam.org/faq/", CheckedAt: "2026-09-12",
		Note: "Official FAQ confirms free desktop software licensed under GNU GPL.",
	},
	"VLC Media Player": {
		Class: LicenseOpenSource, SourceURL: "https://docs.videolan.me/vlc-user/desktop/3.0/en/support/faq/legalconcerns.html", CheckedAt: "2026-09-12",
		Note: "Official VideoLAN documentation identifies GPL v2 licensing.",
	},
	"KeePass 2": {
		Class: LicenseOpenSource, SourceURL: "https://keepass.info/help/v2/license.html", CheckedAt: "2026-09-12",
		Note: "Official KeePass 2 license identifies GNU GPL v2 or later.",
	},
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
