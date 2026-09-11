package catalog

import "strings"

type Platform string
type ExpectedScope string
type InstallerType string
type UninstallStrategy string
type UninstallSupport string
type ResolutionStatus string

const (
	PlatformWindows                    Platform          = "windows"
	ScopeUser                          ExpectedScope     = "user"
	ScopeMachine                       ExpectedScope     = "machine"
	ScopeDynamic                       ExpectedScope     = "dynamic"
	InstallerUnknown                   InstallerType     = "unknown"
	InstallerEXE                       InstallerType     = "exe"
	InstallerMSI                       InstallerType     = "msi"
	InstallerMSIX                      InstallerType     = "msix"
	StrategyWingetUser                 UninstallStrategy = "WingetUser"
	StrategyWingetMachine              UninstallStrategy = "WingetMachine"
	StrategyRegistryUser               UninstallStrategy = "RegistryUser"
	StrategyRegistryMachine            UninstallStrategy = "RegistryMachine"
	StrategyMSIProductCode             UninstallStrategy = "MSIProductCode"
	StrategyMSIXStore                  UninstallStrategy = "MSIXStore"
	StrategyVendorUninstaller          UninstallStrategy = "VendorUninstaller"
	StrategySystemComponentUnsupported UninstallStrategy = "SystemComponentUnsupported"
	StrategyManualOnly                 UninstallStrategy = "ManualOnly"
	StrategyUnresolved                 UninstallStrategy = "Unresolved"
	StrategyRuntimeDetect              UninstallStrategy = "RuntimeDetect"
	SupportAutomatic                   UninstallSupport  = "automatic"
	SupportConditional                 UninstallSupport  = "conditional"
	SupportManual                      UninstallSupport  = "manual"
	SupportUnsupported                 UninstallSupport  = "unsupported"
	ResolutionHardcodedVerified        ResolutionStatus  = "hardcoded_id_verified"
	ResolutionHardcodedUnverified      ResolutionStatus  = "hardcoded_id_unverified"
	ResolutionExactNameVerified        ResolutionStatus  = "exact_name_verified"
	ResolutionRuntimeExact             ResolutionStatus  = "runtime_exact_required"
)

type AppDef struct {
	Name     string
	ID       string
	Category string
}

type Profile struct {
	CanonicalName                       string            `json:"canonical_name"`
	CatalogAliasOf                      string            `json:"catalog_alias_of,omitempty"`
	Platform                            Platform          `json:"platform"`
	WingetID                            string            `json:"winget_id,omitempty"`
	AlternativeIDs                      []string          `json:"alternative_ids,omitempty"`
	DisplayAliases                      []string          `json:"display_aliases"`
	RegistryMatchMode                   string            `json:"registry_match_mode"`
	ForbiddenRegistryVariants           []string          `json:"forbidden_registry_variants"`
	ExpectedScope                       ExpectedScope     `json:"expected_scope"`
	InstallerType                       InstallerType     `json:"installer_type"`
	UninstallStrategy                   UninstallStrategy `json:"uninstall_strategy"`
	UninstallSupport                    UninstallSupport  `json:"uninstall_support"`
	SystemComponent                     bool              `json:"system_component"`
	ResolutionStatus                    ResolutionStatus  `json:"resolution_status"`
	ResolutionEvidence                  string            `json:"resolution_evidence"`
	PackageResolutionSafe               bool              `json:"package_resolution_safe"`
	PhysicalTestRequired                bool              `json:"physical_test_required"`
	DefaultInstallLocationInformational bool              `json:"default_install_location_informational"`
	License                             LicenseMetadata   `json:"license"`
}

var RegistryVariantTokens = []string{"beta", "alpha", "nightly", "canary", "dev", "preview", "cli", "runtime", "updater", "update", "helper", "extension", "sdk", "server", "client", "viewer", "agent", "driver", "portable"}

var verifiedHardcodedIDs = map[string]bool{
	"DBeaver.DBeaver.Community":             true,
	"7zip.7zip":                             true,
	"Anysphere.Cursor":                      true,
	"Audacity.Audacity":                     true,
	"Axosoft.GitKraken":                     true,
	"Balena.Etcher":                         true,
	"BlenderFoundation.Blender":             true,
	"Brave.Brave":                           true,
	"Discord.Discord":                       true,
	"DominikReichl.KeePass":                 true,
	"Dropbox.Dropbox":                       true,
	"ElementLabs.LMStudio":                  true,
	"EpicGames.EpicGamesLauncher":           true,
	"GIMP.GIMP":                             true,
	"Giorgiotani.Peazip":                    true,
	"Git.Git":                               true,
	"GitHub.GitHubDesktop":                  true,
	"GitHub.cli":                            true,
	"GodotEngine.GodotEngine":               true,
	"Google.AndroidStudio":                  true,
	"Google.Chrome":                         true,
	"Google.GoogleDrive":                    true,
	"Greenshot.Greenshot":                   true,
	"Gyan.FFmpeg":                           true,
	"HandBrake.HandBrake":                   true,
	"Inkscape.Inkscape":                     true,
	"Insecure.Nmap":                         true,
	"JetBrains.IntelliJIDEA.Community":      true,
	"JetBrains.Toolbox":                     true,
	"KDE.Kdenlive":                          true,
	"KDE.Krita":                             true,
	"KeePassXCTeam.KeePassXC":               true,
	"Kitware.CMake":                         true,
	"Malwarebytes.Malwarebytes":             true,
	"Meltytech.Shotcut":                     true,
	"Microsoft.Edge":                        true,
	"Microsoft.OneDrive":                    true,
	"Microsoft.PowerToys":                   true,
	"Microsoft.Teams":                       true,
	"Microsoft.VisualStudio.2022.Community": true,
	"Microsoft.VisualStudioCode":            true,
	"Microsoft.WindowsTerminal":             true,
	"Mozilla.Firefox":                       true,
	"Mozilla.Thunderbird":                   true,
	"Notepad++.Notepad++":                   true,
	"OBSProject.OBSStudio":                  true,
	"Ollama.Ollama":                         true,
	"OpenJS.NodeJS.LTS":                     true,
	"OpenShot.OpenShot":                     true,
	"Opera.Opera":                           true,
	"Oracle.VirtualBox":                     true,
	"Piriform.CCleaner":                     true,
	"Postman.Postman":                       true,
	"PuTTY.PuTTY":                           true,
	"Python.Python.3.13":                    true,
	"Rufus.Rufus":                           true,
	"ShareX.ShareX":                         true,
	"Spotify.Spotify":                       true,
	"SumatraPDF.SumatraPDF":                 true,
	"Telegram.TelegramDesktop":              true,
	"TheDocumentFoundation.LibreOffice":     true,
	"Valve.Steam":                           true,
	"Ventoy.Ventoy":                         true,
	"VideoLAN.VLC":                          true,
	"Vivaldi.Vivaldi":                       true,
	"WinDirStat.WinDirStat":                 true,
	"WinSCP.WinSCP":                         true,
	"WiresharkFoundation.Wireshark":         true,
	"Zoom.Zoom":                             true,
	"dotPDN.PaintDotNet":                    true,
	"qBittorrent.qBittorrent":               true,
	"voidtools.Everything":                  true,
}

var exactNameVerified20260905 = map[string]bool{
	"Google Chrome":                true,
	"Mozilla Firefox":              true,
	"Microsoft Edge":               true,
	"Brave":                        true,
	"Opera":                        true,
	"Vivaldi":                      true,
	"Zoom":                         true,
	"Discord":                      true,
	"Microsoft Teams":              true,
	"Pidgin":                       true,
	"Mozilla Thunderbird":          true,
	"Trillian":                     true,
	"Telegram Desktop":             true,
	"iTunes":                       true,
	"VLC Media Player":             true,
	"AIMP":                         true,
	"foobar2000":                   true,
	"Winamp":                       true,
	"Audacity":                     true,
	"GOM Player":                   true,
	"Spotify":                      true,
	"HandBrake":                    true,
	"Krita":                        true,
	"Blender":                      true,
	"Paint.NET":                    true,
	"GIMP":                         true,
	"IrfanView":                    true,
	"XnView":                       true,
	"Inkscape":                     true,
	"FastStone Image Viewer":       true,
	"Greenshot":                    true,
	"ShareX":                       true,
	"OBS Studio":                   true,
	"LibreOffice":                  true,
	"SumatraPDF":                   true,
	"OpenOffice":                   true,
	"Malwarebytes":                 true,
	"SUPERAntiSpyware":             true,
	"qBittorrent":                  true,
	"Dropbox":                      true,
	"Google Drive for Desktop":     true,
	"Microsoft OneDrive":           true,
	"Steam":                        true,
	"Epic Games Launcher":          true,
	"Evernote":                     true,
	"KeePass 2":                    true,
	"Everything":                   true,
	"ImgBurn":                      true,
	"RealVNC Server":               true,
	"RealVNC Viewer":               true,
	"TightVNC":                     true,
	"TeraCopy":                     true,
	"Revo Uninstaller":             true,
	"Launchy":                      true,
	"WinDirStat":                   true,
	"Glary Utilities":              true,
	"InfraRecorder":                true,
	"Open-Shell":                   true,
	"CCleaner":                     true,
	"Microsoft PowerToys":          true,
	"Rufus":                        true,
	"balenaEtcher":                 true,
	"TreeSize Free":                true,
	"Windows Terminal":             true,
	"7-Zip":                        true,
	"PeaZip":                       true,
	"Python 3":                     true,
	"Git":                          true,
	"GitHub Desktop":               true,
	"Node.js":                      true,
	"Notepad++":                    true,
	"WinSCP":                       true,
	"PuTTY":                        true,
	"WinMerge":                     true,
	"Visual Studio Code":           true,
	"Cursor":                       true,
	"Postman":                      true,
	"DBeaver":                      true,
	"Visual Studio Community":      true,
	"IntelliJ IDEA Community":      true,
	"Android Studio":               true,
	"GitKraken":                    true,
	"Amazon Corretto JRE 8":        true,
	"HWiNFO":                       true,
	"CrystalDiskInfo":              true,
	"CrystalDiskMark":              true,
	"FanControl":                   true,
	"MSI Afterburner":              true,
	"Focusrite Control 2":          true,
	"ASIO4ALL":                     true,
	"Godot Engine":                 true,
	"CMake":                        true,
	"Wireshark":                    true,
	"Nmap":                         true,
	"VirtualBox":                   true,
	"Ventoy":                       true,
	"GOG GALAXY":                   true,
	"EA app":                       true,
	"Ubisoft Connect":              true,
	"Rockstar Games Launcher":      true,
	"Prism Launcher":               true,
	"MultiMC":                      true,
	"CurseForge":                   true,
	"Modrinth App":                 true,
	"Playnite":                     true,
	"Sunshine":                     true,
	"Parsec":                       true,
	"Tailscale":                    true,
	"WireGuard":                    true,
	"OpenVPN Connect":              true,
	"Proton VPN":                   true,
	"Windscribe":                   true,
	"TunnelBear":                   true,
	"NetBird":                      true,
	"Radmin VPN":                   true,
	"UltraVNC":                     true,
	"mRemoteNG":                    true,
	"Termius":                      true,
	"NoMachine":                    true,
	"Ollama":                       true,
	"LM Studio":                    true,
	"Jan":                          true,
	"GPT4All":                      true,
	"Open WebUI":                   true,
	"llama.cpp":                    true,
	"Pinokio":                      true,
	"Langflow":                     true,
	"Cherry Studio":                true,
	"Khoj":                         true,
	"ComfyUI":                      true,
	"InvokeAI":                     true,
	"Upscayl":                      true,
	"WampServer":                   true,
	"Laragon":                      true,
	"DDEV":                         true,
	"Podman Desktop":               true,
	"Caddy":                        true,
	"Yarn":                         true,
	"Bun":                          true,
	"Deno":                         true,
	"Python":                       true,
	"MySQL":                        true,
	"MariaDB":                      true,
	"SQLite":                       true,
	"HeidiSQL":                     true,
	"Azure Data Studio":            true,
	"SQLiteStudio":                 true,
	"DB Browser for SQLite":        true,
	"MongoDB Compass":              true,
	"TablePlus":                    true,
	"Beekeeper Studio":             true,
	"DbGate":                       true,
	"Bruno":                        true,
	"Hoppscotch":                   true,
	"HTTPie":                       true,
	"curl":                         true,
	"wget":                         true,
	"SoapUI":                       true,
	"mitmproxy":                    true,
	"Burp Suite Community Edition": true,
	"k6":                           true,
	"Apache JMeter":                true,
	"Stoplight Studio":             true,
	"Yaak":                         true,
	"VSCodium":                     true,
	"Zed":                          true,
	"Code::Blocks":                 true,
	"Geany":                        true,
	"Kate":                         true,
	"Lite XL":                      true,
	"Neovim":                       true,
	"Lapce":                        true,
	"Helix":                        true,
	"Pulsar":                       true,
	"JetBrains Toolbox App":        true,
	"Sourcetree":                   true,
	"TortoiseGit":                  true,
	"TortoiseSVN":                  true,
	"Git Extensions":               true,
	"Lazygit":                      true,
	"GitHub CLI":                   true,
	"FireAlpaca":                   true,
	"MyPaint":                      true,
	"Pinta":                        true,
	"Fotor":                        true,
	"PhotoDemon":                   true,
	"Pixelorama":                   true,
	"LDtk":                         true,
	"OpenSCAD":                     true,
	"MeshLab":                      true,
	"SolveSpace":                   true,
	"Wings 3D":                     true,
	"GDevelop":                     true,
	"darktable":                    true,
	"digiKam":                      true,
	"XnView MP":                    true,
	"ImageGlass":                   true,
	"JPEGView":                     true,
	"ExifTool":                     true,
	"Hugin":                        true,
	"Luminance HDR":                true,
	"Flameshot":                    true,
	"Lightshot":                    true,
	"Streamlabs Desktop":           true,
	"ScreenToGif":                  true,
	"Gyazo":                        true,
	"LICEcap":                      true,
	"Elgato 4K Capture Utility":    true,
	"Elgato Stream Deck":           true,
	"Kdenlive":                     true,
	"Shotcut":                      true,
	"OpenShot":                     true,
	"FFmpeg":                       true,
	"Avidemux":                     true,
	"LosslessCut":                  true,
	"Shutter Encoder":              true,
	"Lightworks":                   true,
	"Olive Video Editor":           true,
	"VSDC Free Video Editor":       true,
	"LMMS":                         true,
	"MuseScore":                    true,
	"Voicemeeter":                  true,
	"Mixxx":                        true,
	"Ocenaudio":                    true,
	"Tenacity":                     true,
	"Waves Central":                true,
	"Line 6 Central":               true,
	"OnlyOffice Desktop Editors":   true,
	"PDF24 Creator":                true,
	"Okular":                       true,
	"Xournal++":                    true,
	"Obsidian":                     true,
	"Notion":                       true,
	"Joplin":                       true,
	"Logseq":                       true,
	"Zotero":                       true,
	"Mendeley Reference Manager":   true,
	"MarkText":                     true,
	"Standard Notes":               true,
	"NanaZip":                      true,
	"Bandizip":                     true,
	"Listary":                      true,
	"Double Commander":             true,
	"FastCopy":                     true,
	"SpaceSniffer":                 true,
	"Sysinternals Suite":           true,
	"Process Explorer":             true,
	"Process Monitor":              true,
	"Autoruns":                     true,
	"TCPView":                      true,
	"Speccy":                       true,
	"RivaTuner Statistics Server":  true,
	"OCCT":                         true,
	"Prime95":                      true,
	"Core Temp":                    true,
	"UNetbootin":                   true,
	"AnyBurn":                      true,
	"Media Creation Tool":          true,
	"NTLite":                       true,
	"QEMU":                         true,
	"Vagrant":                      true,
	"Multipass":                    true,
	"kind":                         true,
	"k3d":                          true,
	"Angry IP Scanner":             true,
	"Advanced IP Scanner":          true,
	"iperf3":                       true,
	"Bitvise SSH Client":           true,
	"Proton Drive":                 true,
	"pCloud Drive":                 true,
	"rclone":                       true,
	"Cyberduck":                    true,
	"Bitwarden":                    true,
	"KeePass":                      true,
	"KeePassXC":                    true,
	"Proton Pass":                  true,
	"Enpass":                       true,
	"Yubico Authenticator":         true,
	"SimpleWall":                   true,
	"Portmaster":                   true,
	"Slack":                        true,
	"Viber":                        true,
	"Element":                      true,
	"Mailspring":                   true,
	"eM Client":                    true,
	"Spark Desktop":                true,
	"LibreWolf":                    true,
	"Waterfox":                     true,
	"Tor Browser":                  true,
	"Zen Browser":                  true,
	"Chromium":                     true,
	"Free Download Manager":        true,
	"JDownloader 2":                true,
	"Xtreme Download Manager":      true,
	"Motrix":                       true,
	"aria2":                        true,
	"Persepolis Download Manager":  true,
	"yt-dlp":                       true,
	"Transmission":                 true,
	"BiglyBT":                      true,
	"Tixati":                       true,
	"Tribler":                      true,
}

var displayAliasOverrides = map[string][]string{
	"Mozilla Firefox":                   {"Mozilla Firefox", "Firefox"},
	"Zoom":                              {"Zoom", "Zoom Workplace"},
	"Mozilla Thunderbird":               {"Mozilla Thunderbird", "Thunderbird"},
	"Google Drive for Desktop":          {"Google Drive for Desktop", "Google Drive"},
	"Microsoft OneDrive":                {"Microsoft OneDrive", "OneDrive"},
	"Microsoft PowerToys":               {"Microsoft PowerToys", "PowerToys"},
	"Visual Studio Code":                {"Visual Studio Code", "Microsoft Visual Studio Code"},
	"IntelliJ IDEA Community":           {"IntelliJ IDEA Community", "IntelliJ IDEA Community Edition"},
	"JetBrains IntelliJ IDEA Community": {"JetBrains IntelliJ IDEA Community", "IntelliJ IDEA Community", "IntelliJ IDEA Community Edition"},
	"Cisco Packet Tracer":               {"Cisco Packet Tracer", "Packet Tracer"},
	"OpenOffice":                        {"OpenOffice", "Apache OpenOffice"},
	"Foxit Reader":                      {"Foxit Reader", "Foxit PDF Reader"},
	"Python 3":                          {"Python 3", "Python 3.13"},
	"Python":                            {"Python", "Python 3", "Python 3.13"},
	"Node.js":                           {"Node.js", "Node.js LTS"},
	"DBeaver":                           {"DBeaver", "DBeaver Community"},
	"KeePass 2":                         {"KeePass 2", "KeePass"},
}

var alternativeIDOverrides = map[string][]string{
	"DBeaver": {"dbeaver.dbeaver"},
}

var aliasOfOverrides = map[string]string{
	"Python":  "Python 3",
	"KeePass": "KeePass 2",
}

var systemComponentOverrides = map[string]bool{
	"Microsoft Edge":      true,
	"OpenSSH for Windows": true,
	"Windows Defender":    true,
}

var msixOverrides = map[string]bool{
	"Windows Terminal": true,
	"Xbox App":         true,
	"Xbox Game Bar":    true,
}

var manualOnlyOverrides = map[string]bool{
	"Pentablet driver / kezelőszoftver": true,
}

func ProfileFor(app AppDef) Profile {
	p := Profile{
		CanonicalName:                       app.Name,
		Platform:                            PlatformWindows,
		WingetID:                            app.ID,
		ExpectedScope:                       ScopeDynamic,
		InstallerType:                       InstallerUnknown,
		UninstallStrategy:                   StrategyRuntimeDetect,
		UninstallSupport:                    SupportConditional,
		PackageResolutionSafe:               true,
		PhysicalTestRequired:                true,
		DefaultInstallLocationInformational: true,
		RegistryMatchMode:                   "strict_alias",
		ForbiddenRegistryVariants:           append([]string(nil), RegistryVariantTokens...),
	}
	if v := aliasOfOverrides[app.Name]; v != "" {
		p.CatalogAliasOf = v
		p.CanonicalName = v
	}
	p.DisplayAliases = aliasesFor(app.Name)
	p.AlternativeIDs = append([]string(nil), alternativeIDOverrides[app.Name]...)
	if systemComponentOverrides[app.Name] {
		p.SystemComponent = true
		p.UninstallStrategy = StrategySystemComponentUnsupported
		p.UninstallSupport = SupportUnsupported
	}
	if msixOverrides[app.Name] {
		p.InstallerType = InstallerMSIX
		p.UninstallStrategy = StrategyMSIXStore
		p.UninstallSupport = SupportConditional
	}
	if manualOnlyOverrides[app.Name] {
		p.UninstallStrategy = StrategyManualOnly
		p.UninstallSupport = SupportManual
	}
	if p.SystemComponent || p.UninstallStrategy == StrategyManualOnly {
		p.PhysicalTestRequired = false
	}
	p.License = LicenseFor(app, p.SystemComponent)
	if app.ID != "" {
		if verifiedHardcodedIDs[app.ID] {
			p.ResolutionStatus = ResolutionHardcodedVerified
			p.ResolutionEvidence = "microsoft/winget-pkgs manifest directory verified 2026-09-06"
		} else {
			p.ResolutionStatus = ResolutionHardcodedUnverified
			p.ResolutionEvidence = "hardcoded ID requires runtime exact verification"
		}
	} else if exactNameVerified20260905[app.Name] {
		p.ResolutionStatus = ResolutionExactNameVerified
		p.ResolutionEvidence = "exact Winget name resolution recorded by v0.6.12 audit on 2026-09-05"
	} else {
		p.ResolutionStatus = ResolutionRuntimeExact
		p.ResolutionEvidence = "no static package ID; exact runtime source query required"
	}
	return p
}

func aliasesFor(name string) []string {
	base := displayAliasOverrides[name]
	if len(base) == 0 {
		base = []string{name}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(base))
	for _, v := range base {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		k := strings.ToLower(v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}

func CandidateQueries(name string) []string {
	for _, app := range Entries {
		if app.Name == name {
			return ProfileFor(app).DisplayAliases
		}
	}
	return aliasesFor(name)
}

func IDsFor(app AppDef) []string {
	p := ProfileFor(app)
	seen := map[string]bool{}
	var out []string
	all := append([]string{p.WingetID}, p.AlternativeIDs...)
	for _, v := range all {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		k := strings.ToLower(v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}
