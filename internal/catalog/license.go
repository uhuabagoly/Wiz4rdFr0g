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
	"Upscayl":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/upscayl/upscayl/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official AGPL-3.0 terms for Upscayl.Upscayl; source SHA256 8486a10c4393cee1c25392769ddd3b2d6c242d6ec7928e1414efff7dfb2f07ef. CI source archive 35686716624; physical lifecycle still required."},
	"Logseq":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/logseq/logseq/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official AGPL-3.0 terms for Logseq.Logseq; source SHA256 2467b8901ba62f7708c479944468a677897472a39ecbda1d23818ecf9538620b. CI source archive 35686716624; physical lifecycle still required."},
	"Standard Notes":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/standardnotes/app/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official AGPL-3.0 terms for StandardNotes.StandardNotes; source SHA256 d8de517917a591daa447d6be28ffb2fac866703e4feb65e86221be9a22d3033a. CI source archive 35686716624; physical lifecycle still required."},
	"Bun":                         {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/oven-sh/bun/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for Oven-sh.Bun; source SHA256 b9caf52728691b4057e371232c221a132883198be2f3d2ddf92c90404c984b1a. CI source archive 35686716624; physical lifecycle still required. Bun MIT with LGPL JavaScriptCore and linked-library licenses."},
	"MyPaint":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/mypaint/mypaint/master/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official GNU General Public License v2.0 terms for MyPaint.MyPaint; source SHA256 ab15fd526bd8dd18a9e77ebc139656bf4d33e97fc7238cd11bf60e2b9b8666c6. CI source archive 35686716624; physical lifecycle still required."},
	"DB Browser for SQLite":       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/sqlitebrowser/sqlitebrowser/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 and GPL-3.0-or-later terms for DBBrowserForSQLite.DBBrowserForSQLite; source SHA256 9a851d93878783b348c7efb177b5f46b09a66115b5c926565ab05f656eb2e173. CI source archive 35686716624; physical lifecycle still required. MPL-2.0/GPL-3.0-or-later dual license, MIT sqlean and third-party resource notices."},
	"k6":                          {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/grafana/k6/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official AGPL-3.0 terms for GrafanaLabs.k6; source SHA256 45bd5efa5e66840e253253f3cc172d07213cb812206cda7bb71cd5b9dbd5a985. CI source archive 35686716624; physical lifecycle still required."},
	"GDevelop":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/4ian/GDevelop/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for GDevelop.GDevelop; source SHA256 0620d885ddbc88e952f99090d767de08671b6a81e5c10900ef5b949531460b92. CI source archive 35686716624; physical lifecycle still required. MIT editor and engine; project name and logo retain trademark ownership."},
	"JPEGView":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/sylikc/jpegview/HEAD/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for sylikc.JPEGView; source SHA256 3834a17ceecdf637633fedab1eb95a49fd12c8524a1c24b1edd131c289d3f4ef. CI source archive 35686716624; physical lifecycle still required."},
	"MultiMC":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/MultiMC/Launcher/develop/COPYING.md", CheckedAt: "2026-09-22", Note: "Reviewed official Dual License (Apache-2.0 and MS-PL) terms for MultiMC.MultiMC; source SHA256 97b06ce7e6c76027b724956f5d06539a0900cbd7c922727f019afe550e9023e5. CI source archive 35686716624; physical lifecycle still required."},
	"yt-dlp":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/yt-dlp/yt-dlp/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official Unlicense terms for yt-dlp.yt-dlp; source SHA256 7e12e5df4bae12cb21581ba157ced20e1986a0508dd10d0e8a4ab9a4cf94e85c. CI source archive 35686716624; physical lifecycle still required."},
	"Transmission":                {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/transmission/transmission/main/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official GPLv3 terms for Transmission.Transmission; source SHA256 9715b55d11caa499d34a6e23498c766fe1ece4ace57777553313e27d688dee78. CI source archive 35686716624; physical lifecycle still required. Publisher permits GPLv2 or GPLv3 with OpenSSL linking exception."},
	"Prism Launcher":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/PrismLauncher/PrismLauncher/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for PrismLauncher.PrismLauncher; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. CI source archive 35686716624; physical lifecycle still required."},
	"Modrinth App":                {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/modrinth/code/main/apps/app/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Modrinth.ModrinthApp; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. CI source archive 35686716624; physical lifecycle still required."},
	"Sunshine":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/LizardByte/Sunshine/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for LizardByte.Sunshine; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. CI source archive 35686716624; physical lifecycle still required."},
	"Tailscale":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/tailscale/tailscale/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official BSD-3-Clause terms for Tailscale.Tailscale; source SHA256 a7ca6186a7963a0a60740f6047760eecd7a0234e8c38bd7e1e0bbcb324bda45b. CI source archive 35686716624; physical lifecycle still required."},
	"Windscribe":                  {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/Windscribe/Desktop-App/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for Windscribe.Windscribe; source SHA256 f9c375a1be4a41f7b70301dd83c91cb89e41567478859b77eef375a52d782505. CI source archive 35686716624; physical lifecycle still required."},
	"llama.cpp":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ggml-org/llama.cpp/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for ggml.llamacpp; source SHA256 94f29bbed6a22c35b992c5c6ebf0e7c92f13b836b90f36f461c9cf2f0f1d010d. CI source archive 35686716624; physical lifecycle still required."},
	"Pinokio":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/pinokiocomputer/pinokio/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for pinokiocomputer.pinokio; source SHA256 454799ad2691bab3af8fda5a9a0d3baf29d517485d36c79a92fc48e07a589881. CI source archive 35686716624; physical lifecycle still required."},
	"DDEV":                        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ddev/ddev/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official Apache-2.0 terms for DDEVFoundation.DDEV; source SHA256 b40930bbcf80744c86c46a12bc9da056641d722716c378f5659b9e555ef833e1. CI source archive 35686716624; physical lifecycle still required."},
	"Caddy":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/caddyserver/caddy/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official Apache-2.0 terms for CaddyServer.Caddy; source SHA256 cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30. CI source archive 35686716624; physical lifecycle still required."},
	"Yarn":                        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/yarnpkg/yarn/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official Distributed under BSD License · Code of Conduct terms for Yarn.Yarn; source SHA256 ae09761b5d27e237571245437653b483bbb85de3cc64b802e8ef47d03aabec1e. CI source archive 35686716624; physical lifecycle still required."},
	"Deno":                        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/denoland/deno/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for DenoLand.Deno; source SHA256 f62497fffecc0852960c8d3e6934b9db86d16396e9b604072e923892cae3a588. CI source archive 35686716624; physical lifecycle still required."},
	"HeidiSQL":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/HeidiSQL/HeidiSQL/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for HeidiSQL.HeidiSQL; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. CI source archive 35686716624; physical lifecycle still required."},
	"DbGate":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/dbgate/dbgate/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for JanProchazka.dbgate; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. CI source archive 35686716624; physical lifecycle still required."},
	"Yaak":                        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/mountain-loop/yaak/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for Yaak.app; source SHA256 e69bfc5b751b76dff095ccddec6c5e1e47c4cef784aca41bdb2ff0835480c4f7. CI source archive 35686716624; physical lifecycle still required."},
	"VSCodium":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/VSCodium/vscodium/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for VSCodium.VSCodium; source SHA256 2fa3f8948a0a17ea30b62845caf1ee8aae8b55b5417273920ca2df2642209e0e. CI source archive 35686716624; physical lifecycle still required."},
	"Zed":                         {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/zed-industries/zed/main/LICENSE-GPL", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for ZedIndustries.Zed; source SHA256 0aa0322ea494da441088e4d9144c6720bf99d9e2e1eefcd29d1f58b19b5c4246. CI source archive 35686716624; physical lifecycle still required."},
	"Geany":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/geany/geany/master/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0-or-later terms for Geany.Geany; source SHA256 ea71ec4ad43b4dd8a32903c17e1058798471f14beaa2209022986ee6a626e9c5. CI source archive 35686716624; physical lifecycle still required."},
	"Lite XL":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/lite-xl/lite-xl/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for LiteXLTeam.LiteXL; source SHA256 87851eccbfcc059f1d1d5bd3ad2849ff031d3679edff9805abf6af32c3837429. CI source archive 35686716624; physical lifecycle still required."},
	"Neovim":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/neovim/neovim/HEAD/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Reviewed official Apache-2.0 AND Vim terms for Neovim.Neovim; source SHA256 de23202ef9a51f5e654034190539739a4abece5505f2ce75e780791106707011. CI source archive 35686716624; physical lifecycle still required."},
	"Lapce":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/lapce/lapce/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official Apache-2.0 terms for Lapce.Lapce; source SHA256 c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4. CI source archive 35686716624; physical lifecycle still required."},
	"Pulsar":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/pulsar-edit/pulsar/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT License terms for Pulsar-Edit.Pulsar; source SHA256 6edc3b6d7a797355a1e14b19edf1388f0ecda5c9059c763457a3bc1953345590. CI source archive 35686716624; physical lifecycle still required."},
	"Pinta":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/PintaProject/Pinta/master/license-mit.txt", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for Pinta.Pinta; source SHA256 422727e784ef57ab534799297b4f9db0cb8e921c59c4ae910043496dabd7718f. CI source archive 35686716624; physical lifecycle still required."},
	"Flameshot":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/flameshot-org/flameshot/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Flameshot.Flameshot; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. CI source archive 35686716624; physical lifecycle still required."},
	"Streamlabs Desktop":          {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/streamlabs/desktop/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Streamlabs.Streamlabs; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. CI source archive 35686716624; physical lifecycle still required."},
	"Xournal++":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/xournalpp/xournalpp/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for Xournal++.Xournal++; source SHA256 f9c375a1be4a41f7b70301dd83c91cb89e41567478859b77eef375a52d782505. CI source archive 35686716624; physical lifecycle still required."},
	"MarkText":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/marktext/marktext/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for MarkText.MarkText; source SHA256 dcba6003d1e8ad62de2e166a2c80ec1ad08cef24f941543ccf8690da7cb89d04. CI source archive 35686716624; physical lifecycle still required."},
	"NanaZip":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/M2Team/NanaZip/HEAD/License.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for M2Team.NanaZip; source SHA256 14c28c4c434e5d104a3d5389a9ec2f5123476820e4293fa1587dacfd793ab6f5. CI source archive 35686716624; physical lifecycle still required. Core includes 7-Zip licenses; icons CC BY-ND 4.0."},
	"Double Commander":            {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/doublecmd/doublecmd/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for alexx2000.DoubleCommander; source SHA256 aef8b4222b79d0dbf6bf17cfff71c90a6a6bb8917a4162abe417b469ed22da2e. CI source archive 35686716624; physical lifecycle still required."},
	"Multipass":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/canonical/multipass/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Canonical.Multipass; source SHA256 fa1ba6aff6dc29b0542f170c0caaa9a13e678d7e3cdb056aeb5d2ae781ba30bc. CI source archive 35686716624; physical lifecycle still required."},
	"kind":                        {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/kubernetes-sigs/kind/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official Apache-2.0 terms for Kubernetes.kind; source SHA256 b40930bbcf80744c86c46a12bc9da056641d722716c378f5659b9e555ef833e1. CI source archive 35686716624; physical lifecycle still required."},
	"Proton Pass":                 {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/protonpass/proton-pass-common/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Proton.ProtonPass; source SHA256 14531a35b0c23125cd4210b4e988bbed9d319d0712b67b4c7788e5a7b884a04d. CI source archive 35686716624; physical lifecycle still required."},
	"Element":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/element-hq/element-desktop/HEAD/LICENSE-GPL-3.0", CheckedAt: "2026-09-22", Note: "Reviewed official AGPL-3.0-only or GPL-3.0-only terms for Element.Element; source SHA256 8b1ba204bb69a0ade2bfcf65ef294a920f6bb361b317dba43c7ef29d96332b9b. CI source archive 35686716624; physical lifecycle still required."},
	"Zen Browser":                 {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/zen-browser/desktop/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for Zen-Team.Zen-Browser; source SHA256 c76f740d1521b9bed9ca7a04ad526c310493c62621b1341d623b431736533b30. CI source archive 35686716624; physical lifecycle still required."},
	"Xtreme Download Manager":     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/subhra74/xdm/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for subhra74.XtremeDownloadManager; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. CI source archive 35686716624; physical lifecycle still required."},
	"Motrix":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/agalwood/Motrix/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for agalwood.Motrix; source SHA256 f60775e705e2c7418665ac2c7f386d28cc2927df98a440ced1703a7ed3ca86b7. CI source archive 35686716624; physical lifecycle still required."},
	"aria2":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/aria2/aria2/master/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0-or-later terms for aria2.aria2; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. CI source archive 35686716624; physical lifecycle still required."},
	"Persepolis Download Manager": {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/persepolisdm/persepolis/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for PersepolisDownloadManager.Persepolis; source SHA256 589ed823e9a84c56feb95ac58e7cf384626b9cbf4fda2a907bc36e103de1bad2. CI source archive 35686716624; physical lifecycle still required."},
	"Mozilla Firefox":             {Class: LicenseOpenSource, SourceURL: "https://www.mozilla.org/en-US/MPL/2.0/", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for Mozilla.Firefox; source SHA256 79fefbb5ce750421056e8639e83b055fa97deb02407617d4c0f9d50cfa0cfd32. Physical lifecycle still required."},
	"Mozilla Thunderbird":         {Class: LicenseOpenSource, SourceURL: "https://www.mozilla.org/en-US/MPL/2.0/", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for Mozilla.Thunderbird; source SHA256 79fefbb5ce750421056e8639e83b055fa97deb02407617d4c0f9d50cfa0cfd32. Physical lifecycle still required."},
	"LibreOffice":                 {Class: LicenseOpenSource, SourceURL: "https://www.libreoffice.org/licenses/", CheckedAt: "2026-09-22", Note: "Reviewed official MPL-2.0 terms for TheDocumentFoundation.LibreOffice; source SHA256 fc8c0e5c724cf292e03af59d3241fccc8b29f1120f54c731a46240fcea62ee3b. Physical lifecycle still required."},
	"PeaZip":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/peazip/PeaZip/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official LGPL-3.0 terms for Giorgiotani.Peazip; source SHA256 a853c2ffec17057872340eee242ae4d96cbf2b520ae27d903e1b2fef1a5f9d1c. Physical lifecycle still required."},
	"Python 3":                    {Class: LicenseOpenSource, SourceURL: "https://docs.python.org/3/license.html", CheckedAt: "2026-09-22", Note: "Reviewed official PSF-2.0 terms for Python.Python.3.13; source SHA256 287275506de7ece16530519cd9474b1bc5c63eb8b6010d0f713359ec18931a8a. Physical lifecycle still required."},
	"PuTTY":                       {Class: LicenseOpenSource, SourceURL: "https://www.chiark.greenend.org.uk/~sgtatham/putty/licence.html", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for PuTTY.PuTTY; source SHA256 643a9142655b672ef53f729dded917cff6e0f9dfa22d32629ff5ec137e15e3c5. Physical lifecycle still required."},
	"VirtualBox":                  {Class: LicenseOpenSource, SourceURL: "https://www.virtualbox.org/wiki/Licensing_FAQ", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0-only terms for Oracle.VirtualBox; source SHA256 02563e1fcebacf1917e17e9a2d51dd33365c05cfb6f78099e9fb4820aa39e3ff. Physical lifecycle still required. Covers base package only, not the Extension Pack."},
	"Playnite":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/JosefNemec/Playnite/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official MIT terms for Playnite.Playnite; source SHA256 10c0871ee7dd3ac9f6ef53097a173113229edd1b23e9218d952287889b2b84eb. Physical lifecycle still required."},
	"Python":                      {Class: LicenseOpenSource, SourceURL: "https://docs.python.org/3/license.html", CheckedAt: "2026-09-22", Note: "Reviewed official PSF-2.0 terms for Python.Python.3.13; source SHA256 287275506de7ece16530519cd9474b1bc5c63eb8b6010d0f713359ec18931a8a. Physical lifecycle still required."},
	"FFmpeg":                      {Class: LicenseOpenSource, SourceURL: "https://www.ffmpeg.org/legal.html", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-3.0 terms for Gyan.FFmpeg; source SHA256 cd51dc25f9e3200b1e3953aee036f499ed51f01f7ba30563873186cb3f870eda. Physical lifecycle still required."},
	"KeePass":                     {Class: LicenseOpenSource, SourceURL: "https://keepass.info/help/v2/license.html", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0 terms for DominikReichl.KeePass; source SHA256 c5f8a4e39f723cb1ead3175f0f5b33a51feabf56ad8e5977562f792ace061995. Physical lifecycle still required."},
	"KeePassXC":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/keepassxreboot/keepassxc/HEAD/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official GPL-2.0-or-later terms for KeePassXCTeam.KeePassXC; source SHA256 f4cf558763d725e47a55da6f32735b3f0b3d69184870f921015b3c54b58bfb36. Physical lifecycle still required."},
	"Brave":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/brave/brave-browser/master/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MPL-2.0 terms for Brave.Brave; source SHA256 3f3d9e0024b1921b067d6f7f88deb4a60cbe7a78e76c64e3f1d7fc3b779b9d04. Source collection CI 35685779903; physical lifecycle still required."},
	"SumatraPDF":                  {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/sumatrapdfreader/sumatrapdf/3.6rel/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for SumatraPDF.SumatraPDF; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. Source collection CI 35685779903; physical lifecycle still required."},
	"Everything":                  {Class: LicenseOpenSource, SourceURL: "https://www.voidtools.com/License.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for voidtools.Everything; source SHA256 c13d19adcbfd5d07e9512de9df99956a3423399ed1fadc5fd33186697ad8df2f. Source collection CI 35685779903; physical lifecycle still required."},
	"WinDirStat":                  {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/windirstat/windirstat/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-2.0 terms for WinDirStat.WinDirStat; source SHA256 5d05a329bdd65bc1212fda57942ea90a2a810ff7ce192dc38d7042a55caa4a28. Source collection CI 35685779903; physical lifecycle still required."},
	"Rufus":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/pbatard/rufus/HEAD/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0-or-later terms for Rufus.Rufus; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. Source collection CI 35685779903; physical lifecycle still required."},
	"balenaEtcher":                {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/balena-io/etcher/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project Apache-2.0 terms for Balena.Etcher; source SHA256 39445b459f86621683f9731fb6a7d070819dc379840e8ef52f62bb1c68942291. Source collection CI 35685779903; physical lifecycle still required."},
	"Windows Terminal":            {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/microsoft/terminal/main/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for Microsoft.WindowsTerminal; source SHA256 5d177f23ecfeb0ea8e050b6a5a16355e1ae9a0b286436ca8f83ed08b3795be6b. Source collection CI 35685779903; physical lifecycle still required."},
	"WinSCP":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/winscp/winscp/HEAD/license.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0-or-later terms for WinSCP.WinSCP; source SHA256 38493e2a77946bb116831c1a394b5f9d073b8490be20ebc6b5e42df64237dbaa. Source collection CI 35685779903; physical lifecycle still required."},
	"DBeaver":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/dbeaver/dbeaver/HEAD/LICENSE.md", CheckedAt: "2026-09-22", Note: "Reviewed official project Apache-2.0 terms for DBeaver.DBeaver.Community; source SHA256 3574a35d6ceb85fe03ed8dee865ebfcfbbd7c6de6899e890966f437ffe177373. Source collection CI 35685779903; physical lifecycle still required."},
	"Godot Engine":                {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/godotengine/godot/HEAD/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for GodotEngine.GodotEngine; source SHA256 b0435e3b3e4e55238f05f4b306f30524a1b2e20147810d436eaa554fa6855c80. Source collection CI 35685779903; physical lifecycle still required."},
	"CMake":                       {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/Kitware/CMake/HEAD/LICENSE.rst", CheckedAt: "2026-09-22", Note: "Reviewed official project BSD-3-Clause terms for Kitware.CMake; source SHA256 4382e7c1879ac90e3f101a395d23846fa4dbcaa1eed7265b43681e348754825d. Source collection CI 35685779903; physical lifecycle still required."},
	"Ventoy":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ventoy/Ventoy/master/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for Ventoy.Ventoy; source SHA256 3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986. Source collection CI 35685779903; physical lifecycle still required."},
	"Ollama":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ollama/ollama/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for Ollama.Ollama; source SHA256 5934ed2ce0d15154bcdb9c85203210abac0da4314af34081e36df4599f90b226. Source collection CI 35685779903; physical lifecycle still required."},
	"GitHub CLI":                  {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/cli/cli/HEAD/LICENSE", CheckedAt: "2026-09-22", Note: "Reviewed official project MIT terms for GitHub.cli; source SHA256 6da4adc42392c8485e40b4251c7e332fc3352df1947c9ffade71dd60b14a7a4f. Source collection CI 35685779903; physical lifecycle still required."},
	"Shotcut":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/mltframework/shotcut/HEAD/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for Meltytech.Shotcut; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. Source collection CI 35685779903; physical lifecycle still required."},
	"OpenShot":                    {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/OpenShot/openshot-qt/HEAD/COPYING", CheckedAt: "2026-09-22", Note: "Reviewed official project GPL-3.0 terms for OpenShot.OpenShot; source SHA256 667da052f7e5e0c1e49b6921a1811b536ff6e8cfb9719e628dc08dae487925d2. Source collection CI 35685779903; physical lifecycle still required."},
	"GIMP":                        {Class: LicenseOpenSource, SourceURL: "https://www.gimp.org/about/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. No physical PASS implied."},
	"HandBrake":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/HandBrake/HandBrake/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. No physical PASS implied."},
	"OBS Studio":                  {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/obsproject/obs-studio/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8177f97513213526df2cf6184d8ff986c675afb514d4e68a404010521b880643. No physical PASS implied."},
	"ShareX":                      {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/ShareX/ShareX/master/LICENSE.txt", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 c53a65c2fd561c87eaabf1072ef5dcab8653042bc15308465f52413585eb6271. No physical PASS implied."},
	"Greenshot":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/greenshot/greenshot/develop/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903. No physical PASS implied."},
	"qBittorrent":                 {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/qbittorrent/qBittorrent/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 e675cd856f9817474455200ba7e6f5b7cc42d6598a5eecbbbdaa0e6fd304d6b7. No physical PASS implied."},
	"Telegram Desktop":            {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/telegramdesktop/tdesktop/dev/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 a728b65d02eeaf6b50c075a33b76107c652c3e5a12ff67f53682c83d47a23e1e. No physical PASS implied."},
	"Git":                         {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/git/git/master/COPYING", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 5b2198d1645f767585e8a88ac0499b04472164c0d2da22e75ecf97ef443ab32e. No physical PASS implied."},
	"Microsoft PowerToys":         {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/microsoft/PowerToys/main/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 1eb3cbe7f022addbfc6d65cb2e39aca5b24333942ed843aa10c03befd71ee225. No physical PASS implied."},
	"Notepad++":                   {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/notepad-plus-plus/notepad-plus-plus/master/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 2b94f58d89424af06d1a8e16775774757f1ecfb678203c3439af037a24f35dc6. No physical PASS implied."},
	"GitHub Desktop":              {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/desktop/desktop/development/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 891d678cd6aa67c0712f663b5fee690f24d11d360795300814f7bf2eb91ba530. No physical PASS implied."},
	"Node.js":                     {Class: LicenseOpenSource, SourceURL: "https://raw.githubusercontent.com/nodejs/node/main/LICENSE", CheckedAt: "2026-09-22", Note: "Official project license retrieved and reviewed; source SHA256 37110192cd7621a80510e2f2630ae08f0420f97257d4ae1c4d51c11261f1f4b7. No physical PASS implied."},
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
