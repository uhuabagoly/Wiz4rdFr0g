//go:build linux

package main

/*
#cgo LDFLAGS: -Wl,-rpath,/lib/x86_64-linux-gnu -l:libgtk-3.so.0 -l:libgobject-2.0.so.0 -l:libglib-2.0.so.0
#include <stdlib.h>
#include <stdint.h>

typedef void GtkWidget;
typedef void GtkTextBuffer;
typedef void gpointer;
typedef int gboolean;

typedef void (*GCallback)(void);

extern void gtk_init(int*, char***);
extern gboolean gtk_init_check(int*, char***);
extern GtkWidget* gtk_window_new(int);
extern void gtk_window_set_title(void*, const char*);
extern void gtk_window_set_default_size(void*, int, int);
extern void gtk_window_set_resizable(void*, gboolean);
extern gboolean gtk_window_set_icon_from_file(void*, const char*, void**);
extern GtkWidget* gtk_box_new(int, int);
extern GtkWidget* gtk_grid_new(void);
extern void gtk_grid_set_row_spacing(void*, unsigned int);
extern void gtk_grid_set_column_spacing(void*, unsigned int);
extern void gtk_grid_attach(void*, void*, int, int, int, int);
extern void gtk_container_add(void*, void*);
extern void gtk_container_set_border_width(void*, unsigned int);
extern void gtk_box_pack_start(void*, void*, gboolean, gboolean, unsigned int);
extern void gtk_box_pack_end(void*, void*, gboolean, gboolean, unsigned int);
extern GtkWidget* gtk_label_new(const char*);
extern void gtk_label_set_text(void*, const char*);
extern void gtk_label_set_xalign(void*, float);
extern GtkWidget* gtk_entry_new(void);
extern const char* gtk_entry_get_text(void*);
extern void gtk_entry_set_text(void*, const char*);
extern void gtk_entry_set_placeholder_text(void*, const char*);
extern void gtk_editable_set_editable(void*, gboolean);
extern GtkWidget* gtk_button_new_with_label(const char*);
extern void gtk_button_set_label(void*, const char*);
extern GtkWidget* gtk_check_button_new(void);
extern gboolean gtk_toggle_button_get_active(void*);
extern void gtk_toggle_button_set_active(void*, gboolean);
extern GtkWidget* gtk_combo_box_text_new(void);
extern void gtk_combo_box_text_append_text(void*, const char*);
extern void gtk_combo_box_text_remove_all(void*);
extern char* gtk_combo_box_text_get_active_text(void*);
extern void gtk_combo_box_set_active(void*, int);
extern GtkWidget* gtk_scrolled_window_new(void*, void*);
extern void gtk_scrolled_window_set_policy(void*, int, int);
extern GtkWidget* gtk_text_view_new(void);
extern void gtk_text_view_set_editable(void*, gboolean);
extern void gtk_text_view_set_cursor_visible(void*, gboolean);
extern void gtk_text_view_set_monospace(void*, gboolean);
extern GtkTextBuffer* gtk_text_view_get_buffer(void*);
extern void gtk_text_buffer_set_text(void*, const char*, int);
extern void gtk_text_buffer_insert_at_cursor(void*, const char*, int);
extern void gtk_widget_set_size_request(void*, int, int);
extern void gtk_widget_set_sensitive(void*, gboolean);
extern void gtk_widget_show_all(void*);
extern void gtk_widget_show(void*);
extern void gtk_widget_hide(void*);
extern void gtk_widget_destroy(void*);
extern void gtk_main(void);
extern void gtk_main_quit(void);
extern unsigned long g_signal_connect_data(void*, const char*, GCallback, void*, void*, int);
extern unsigned int g_idle_add(gboolean (*function)(void*), void* data);
extern void g_free(void*);

extern void goClicked(int);
extern void goSearchChanged(void);
extern void goDrainUI(void);

static void clicked_cb(void* w, void* data) { goClicked((int)(intptr_t)data); }
static void changed_cb(void* w, void* data) { goSearchChanged(); }
static void destroy_cb(void* w, void* data) { gtk_main_quit(); }
static gboolean idle_cb(void* data) { goDrainUI(); return 0; }
static void connect_clicked(void* w, int id) { g_signal_connect_data(w, "clicked", (GCallback)clicked_cb, (void*)(intptr_t)id, NULL, 0); }
static void connect_changed(void* w) { g_signal_connect_data(w, "changed", (GCallback)changed_cb, NULL, NULL, 0); }
static void connect_destroy(void* w) { g_signal_connect_data(w, "destroy", (GCallback)destroy_cb, NULL, NULL, 0); }
static void schedule_idle(void) { g_idle_add(idle_cb, NULL); }
*/
import "C"

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"wiz4rdfr0g.local/fullcatalog/internal/linuxpkg"
)

const (
	appTitle    = "Wiz4rd Fr0g"
	appVersion  = "0.6.23-linux-full-catalog"
	visibleRows = 16

	idPrev     = 100
	idNext     = 101
	idNextPage = 102
	idBack     = 103
	idInstall  = 104
	idCancel   = 105
	idFinish   = 106
	idClose    = 107
	idRemove   = 108
)

type linuxCandidate struct {
	Name, Category           string
	Apt, Dnf, Pacman, Zypper []string
	Flatpak                  []string
}

type linuxApp struct {
	linuxCandidate
	Provider      string
	Package       string
	FlatpakRemote string
	InstallPath   string
}

type rowUI struct {
	catalogIndex             int
	check, name, combo, path unsafe.Pointer
}

type appState struct {
	Selected bool
	Version  string
	Versions []string
	Loading  bool
}

type operationResult struct {
	OK  bool
	Err string
}

var candidates = []linuxCandidate{
	{"Mozilla Firefox", "Böngészők", []string{"firefox-esr", "firefox"}, []string{"firefox"}, []string{"firefox"}, []string{"MozillaFirefox"}, []string{"org.mozilla.firefox"}},
	{"Chromium", "Böngészők", []string{"chromium", "chromium-browser"}, []string{"chromium"}, []string{"chromium"}, []string{"chromium"}, []string{"org.chromium.Chromium"}},
	{"Brave", "Böngészők", nil, nil, nil, nil, []string{"com.brave.Browser"}},
	{"Google Chrome", "Böngészők", []string{"google-chrome-stable"}, nil, nil, nil, []string{"com.google.Chrome"}},
	{"Vivaldi", "Böngészők", []string{"vivaldi-stable"}, nil, nil, nil, []string{"com.vivaldi.Vivaldi"}},
	{"Opera", "Böngészők", []string{"opera-stable"}, nil, nil, nil, []string{"com.opera.Opera"}},
	{"Tor Browser Launcher", "Böngészők", []string{"torbrowser-launcher"}, []string{"torbrowser-launcher"}, []string{"torbrowser-launcher"}, nil, []string{"com.github.micahflee.torbrowser-launcher"}},

	{"Discord", "Kommunikáció", nil, nil, nil, nil, []string{"com.discordapp.Discord"}},
	{"Zoom", "Kommunikáció", nil, nil, nil, nil, []string{"us.zoom.Zoom"}},
	{"Telegram Desktop", "Kommunikáció", []string{"telegram-desktop"}, []string{"telegram-desktop"}, []string{"telegram-desktop"}, []string{"telegram-desktop"}, []string{"org.telegram.desktop"}},
	{"Signal Desktop", "Kommunikáció", nil, nil, nil, nil, []string{"org.signal.Signal"}},
	{"Element", "Kommunikáció", nil, nil, nil, nil, []string{"im.riot.Riot"}},
	{"Mozilla Thunderbird", "Kommunikáció", []string{"thunderbird"}, []string{"thunderbird"}, []string{"thunderbird"}, []string{"MozillaThunderbird"}, []string{"org.mozilla.Thunderbird"}},
	{"Pidgin", "Kommunikáció", []string{"pidgin"}, []string{"pidgin"}, []string{"pidgin"}, []string{"pidgin"}, []string{"im.pidgin.Pidgin"}},

	{"VLC Media Player", "Média", []string{"vlc"}, []string{"vlc"}, []string{"vlc"}, []string{"vlc"}, []string{"org.videolan.VLC"}},
	{"Audacity", "Média", []string{"audacity"}, []string{"audacity"}, []string{"audacity"}, []string{"audacity"}, []string{"org.audacityteam.Audacity"}},
	{"HandBrake", "Média", []string{"handbrake"}, []string{"HandBrake-gui"}, []string{"handbrake"}, []string{"handbrake-gtk"}, []string{"fr.handbrake.ghb"}},
	{"OBS Studio", "Média", []string{"obs-studio"}, []string{"obs-studio"}, []string{"obs-studio"}, []string{"obs-studio"}, []string{"com.obsproject.Studio"}},
	{"Kdenlive", "Média", []string{"kdenlive"}, []string{"kdenlive"}, []string{"kdenlive"}, []string{"kdenlive"}, []string{"org.kde.kdenlive"}},
	{"Shotcut", "Média", []string{"shotcut"}, []string{"shotcut"}, []string{"shotcut"}, []string{"shotcut"}, []string{"org.shotcut.Shotcut"}},
	{"OpenShot", "Média", []string{"openshot-qt"}, []string{"openshot"}, []string{"openshot"}, []string{"openshot-qt"}, []string{"org.openshot.OpenShot"}},
	{"FFmpeg", "Média", []string{"ffmpeg"}, []string{"ffmpeg"}, []string{"ffmpeg"}, []string{"ffmpeg"}, nil},
	{"MuseScore", "Média", []string{"musescore3", "musescore"}, []string{"musescore"}, []string{"musescore"}, []string{"musescore"}, []string{"org.musescore.MuseScore"}},
	{"LMMS", "Média", []string{"lmms"}, []string{"lmms"}, []string{"lmms"}, []string{"lmms"}, []string{"io.lmms.LMMS"}},
	{"Ardour", "Média", []string{"ardour"}, []string{"ardour"}, []string{"ardour"}, []string{"ardour"}, []string{"org.ardour.Ardour"}},

	{"Krita", "Grafika", []string{"krita"}, []string{"krita"}, []string{"krita"}, []string{"krita"}, []string{"org.kde.krita"}},
	{"Blender", "Grafika", []string{"blender"}, []string{"blender"}, []string{"blender"}, []string{"blender"}, []string{"org.blender.Blender"}},
	{"GIMP", "Grafika", []string{"gimp"}, []string{"gimp"}, []string{"gimp"}, []string{"gimp"}, []string{"org.gimp.GIMP"}},
	{"Inkscape", "Grafika", []string{"inkscape"}, []string{"inkscape"}, []string{"inkscape"}, []string{"inkscape"}, []string{"org.inkscape.Inkscape"}},
	{"MyPaint", "Grafika", []string{"mypaint"}, []string{"mypaint"}, []string{"mypaint"}, []string{"mypaint"}, []string{"org.mypaint.MyPaint"}},
	{"Pinta", "Grafika", []string{"pinta"}, []string{"pinta"}, []string{"pinta"}, []string{"pinta"}, []string{"com.github.PintaProject.Pinta"}},
	{"darktable", "Grafika", []string{"darktable"}, []string{"darktable"}, []string{"darktable"}, []string{"darktable"}, []string{"org.darktable.Darktable"}},
	{"RawTherapee", "Grafika", []string{"rawtherapee"}, []string{"rawtherapee"}, []string{"rawtherapee"}, []string{"rawtherapee"}, []string{"com.rawtherapee.RawTherapee"}},
	{"digiKam", "Grafika", []string{"digikam"}, []string{"digikam"}, []string{"digikam"}, []string{"digikam"}, []string{"org.kde.digikam"}},
	{"FreeCAD", "Grafika / CAD", []string{"freecad"}, []string{"freecad"}, []string{"freecad"}, []string{"FreeCAD"}, []string{"org.freecad.FreeCAD"}},
	{"OpenSCAD", "Grafika / CAD", []string{"openscad"}, []string{"openscad"}, []string{"openscad"}, []string{"openscad"}, []string{"org.openscad.OpenSCAD"}},
	{"Scribus", "Grafika", []string{"scribus"}, []string{"scribus"}, []string{"scribus"}, []string{"scribus"}, []string{"net.scribus.Scribus"}},

	{"LibreOffice", "Dokumentumok", []string{"libreoffice"}, []string{"libreoffice"}, []string{"libreoffice-fresh", "libreoffice-still"}, []string{"libreoffice"}, []string{"org.libreoffice.LibreOffice"}},
	{"Okular", "Dokumentumok", []string{"okular"}, []string{"okular"}, []string{"okular"}, []string{"okular"}, []string{"org.kde.okular"}},
	{"Evince", "Dokumentumok", []string{"evince"}, []string{"evince"}, []string{"evince"}, []string{"evince"}, []string{"org.gnome.Evince"}},
	{"Xournal++", "Dokumentumok", []string{"xournalpp"}, []string{"xournalpp"}, []string{"xournalpp"}, []string{"xournalpp"}, []string{"com.github.xournalpp.xournalpp"}},
	{"Zotero", "Dokumentumok", nil, nil, nil, nil, []string{"org.zotero.Zotero"}},

	{"7-Zip / p7zip", "Tömörítés", []string{"7zip", "p7zip-full"}, []string{"p7zip", "p7zip-plugins"}, []string{"7zip", "p7zip"}, []string{"p7zip-full"}, nil},
	{"PeaZip", "Tömörítés", nil, nil, nil, nil, []string{"io.github.peazip.PeaZip"}},
	{"qBittorrent", "Torrent", []string{"qbittorrent"}, []string{"qbittorrent"}, []string{"qbittorrent"}, []string{"qbittorrent"}, []string{"org.qbittorrent.qBittorrent"}},
	{"Transmission", "Torrent", []string{"transmission-gtk"}, []string{"transmission-gtk"}, []string{"transmission-gtk"}, []string{"transmission-gtk"}, []string{"com.transmissionbt.Transmission"}},

	{"Git", "Fejlesztés", []string{"git"}, []string{"git"}, []string{"git"}, []string{"git"}, nil},
	{"Visual Studio Code", "Fejlesztés", []string{"code"}, []string{"code"}, []string{"code"}, nil, []string{"com.visualstudio.code"}},
	{"VSCodium", "Fejlesztés", []string{"codium"}, []string{"codium"}, []string{"vscodium"}, nil, []string{"com.vscodium.codium"}},
	{"Node.js", "Fejlesztés", []string{"nodejs"}, []string{"nodejs"}, []string{"nodejs"}, []string{"nodejs"}, nil},
	{"Python 3", "Fejlesztés", []string{"python3"}, []string{"python3"}, []string{"python"}, []string{"python3"}, nil},
	{"Go", "Fejlesztés", []string{"golang-go"}, []string{"golang"}, []string{"go"}, []string{"go"}, nil},
	{"Rust", "Fejlesztés", []string{"rustc", "cargo"}, []string{"rust", "cargo"}, []string{"rust"}, []string{"rust"}, nil},
	{"CMake", "Fejlesztés", []string{"cmake"}, []string{"cmake"}, []string{"cmake"}, []string{"cmake"}, nil},
	{"Eclipse IDE", "Fejlesztés", []string{"eclipse"}, []string{"eclipse"}, []string{"eclipse-java"}, nil, []string{"org.eclipse.Java"}},
	{"Apache NetBeans", "Fejlesztés", []string{"netbeans"}, []string{"netbeans"}, []string{"netbeans"}, []string{"netbeans"}, []string{"org.apache.netbeans"}},
	{"Android Studio", "Fejlesztés", nil, nil, nil, nil, []string{"com.google.AndroidStudio"}},
	{"IntelliJ IDEA Community", "Fejlesztés", nil, nil, nil, nil, []string{"com.jetbrains.IntelliJ-IDEA-Community"}},
	{"DBeaver Community", "Fejlesztés / Adatbázis", []string{"dbeaver-ce"}, []string{"dbeaver"}, []string{"dbeaver"}, nil, []string{"io.dbeaver.DBeaverCommunity"}},
	{"SQLite Browser", "Fejlesztés / Adatbázis", []string{"sqlitebrowser"}, []string{"sqlitebrowser"}, []string{"sqlitebrowser"}, []string{"sqlitebrowser"}, []string{"org.sqlitebrowser.sqlitebrowser"}},
	{"pgAdmin 4", "Fejlesztés / Adatbázis", []string{"pgadmin4"}, []string{"pgadmin4"}, nil, nil, nil},
	{"PostgreSQL", "Adatbázis", []string{"postgresql"}, []string{"postgresql-server"}, []string{"postgresql"}, []string{"postgresql-server"}, nil},
	{"MariaDB", "Adatbázis", []string{"mariadb-server"}, []string{"mariadb-server"}, []string{"mariadb"}, []string{"mariadb"}, nil},
	{"SQLite", "Adatbázis", []string{"sqlite3"}, []string{"sqlite"}, []string{"sqlite"}, []string{"sqlite3"}, nil},
	{"Neovim", "Fejlesztés", []string{"neovim"}, []string{"neovim"}, []string{"neovim"}, []string{"neovim"}, nil},
	{"Vim", "Fejlesztés", []string{"vim"}, []string{"vim-enhanced"}, []string{"vim"}, []string{"vim"}, nil},
	{"Emacs", "Fejlesztés", []string{"emacs"}, []string{"emacs"}, []string{"emacs"}, []string{"emacs"}, nil},
	{"Geany", "Fejlesztés", []string{"geany"}, []string{"geany"}, []string{"geany"}, []string{"geany"}, []string{"org.geany.Geany"}},

	{"Steam", "Játék", []string{"steam-installer", "steam"}, []string{"steam"}, []string{"steam"}, []string{"steam"}, []string{"com.valvesoftware.Steam"}},
	{"Heroic Games Launcher", "Játék", nil, nil, nil, nil, []string{"com.heroicgameslauncher.hgl"}},
	{"Lutris", "Játék", []string{"lutris"}, []string{"lutris"}, []string{"lutris"}, []string{"lutris"}, []string{"net.lutris.Lutris"}},
	{"Prism Launcher", "Játék", []string{"prismlauncher"}, []string{"prismlauncher"}, []string{"prismlauncher"}, nil, []string{"org.prismlauncher.PrismLauncher"}},
	{"itch.io App", "Játék", nil, nil, nil, nil, []string{"io.itch.itch"}},

	{"Wireshark", "Hálózat / IT", []string{"wireshark"}, []string{"wireshark"}, []string{"wireshark-qt"}, []string{"wireshark"}, []string{"org.wireshark.Wireshark"}},
	{"Nmap", "Hálózat / IT", []string{"nmap"}, []string{"nmap"}, []string{"nmap"}, []string{"nmap"}, nil},
	{"PuTTY", "Hálózat / IT", []string{"putty"}, []string{"putty"}, []string{"putty"}, []string{"putty"}, nil},
	{"FileZilla", "Hálózat / IT", []string{"filezilla"}, []string{"filezilla"}, []string{"filezilla"}, []string{"filezilla"}, []string{"org.filezillaproject.Filezilla"}},
	{"Remmina", "Hálózat / IT", []string{"remmina"}, []string{"remmina"}, []string{"remmina"}, []string{"remmina"}, []string{"org.remmina.Remmina"}},
	{"WireGuard Tools", "Hálózat / IT", []string{"wireguard-tools"}, []string{"wireguard-tools"}, []string{"wireguard-tools"}, []string{"wireguard-tools"}, nil},
	{"OpenVPN", "Hálózat / IT", []string{"openvpn"}, []string{"openvpn"}, []string{"openvpn"}, []string{"openvpn"}, nil},
	{"iperf3", "Hálózat / IT", []string{"iperf3"}, []string{"iperf3"}, []string{"iperf3"}, []string{"iperf3"}, nil},
	{"GNS3", "Hálózat / IT", []string{"gns3-gui"}, nil, nil, nil, nil},

	{"VirtualBox", "Virtualizáció", []string{"virtualbox"}, []string{"VirtualBox"}, []string{"virtualbox"}, []string{"virtualbox"}, nil},
	{"QEMU", "Virtualizáció", []string{"qemu-system"}, []string{"qemu-system-x86"}, []string{"qemu-full"}, []string{"qemu"}, nil},
	{"virt-manager", "Virtualizáció", []string{"virt-manager"}, []string{"virt-manager"}, []string{"virt-manager"}, []string{"virt-manager"}, []string{"org.virt_manager.virt-manager"}},
	{"GNOME Boxes", "Virtualizáció", []string{"gnome-boxes"}, []string{"gnome-boxes"}, []string{"gnome-boxes"}, []string{"gnome-boxes"}, []string{"org.gnome.Boxes"}},
	{"Podman", "Virtualizáció", []string{"podman"}, []string{"podman"}, []string{"podman"}, []string{"podman"}, nil},

	{"Syncthing", "Felhő / Szinkron", []string{"syncthing"}, []string{"syncthing"}, []string{"syncthing"}, []string{"syncthing"}, nil},
	{"rclone", "Felhő / Szinkron", []string{"rclone"}, []string{"rclone"}, []string{"rclone"}, []string{"rclone"}, nil},
	{"Nextcloud Desktop", "Felhő / Szinkron", []string{"nextcloud-desktop"}, []string{"nextcloud-client"}, []string{"nextcloud-client"}, []string{"nextcloud-desktop"}, []string{"com.nextcloud.desktopclient.nextcloud"}},

	{"KeePassXC", "Biztonság", []string{"keepassxc"}, []string{"keepassxc"}, []string{"keepassxc"}, []string{"keepassxc"}, []string{"org.keepassxc.KeePassXC"}},
	{"ClamAV", "Biztonság", []string{"clamav"}, []string{"clamav"}, []string{"clamav"}, []string{"clamav"}, nil},
	{"VeraCrypt", "Biztonság", []string{"veracrypt"}, nil, nil, nil, nil},

	{"Ollama", "Lokális AI", nil, nil, nil, nil, nil},
}

var (
	window, page1, page2, page3                                                unsafe.Pointer
	searchEntry, countLabel, prevBtn, nextBtn, nextPageBtn                     unsafe.Pointer
	summaryView, logView, backBtn, installBtn, removeBtn, cancelBtn, finishBtn unsafe.Pointer
	finishView, closeBtn                                                       unsafe.Pointer
	rows                                                                       []*rowUI
	apps                                                                       []linuxApp
	states                                                                     []appState
	filtered                                                                   []int
	page                                                                       int
	distroName, packageManager                                                 string
	packageSet                                                                 map[string]bool
	flatpakSet                                                                 map[string]string
	uiMu                                                                       sync.Mutex
	uiQueue                                                                    []func()
	opMu                                                                       sync.Mutex
	opCancel                                                                   context.CancelFunc
	operationResults                                                           map[int]operationResult
	currentOperation                                                           string
)

func cstr(s string) *C.char { return C.CString(s) }
func setLabel(w unsafe.Pointer, s string) {
	cs := cstr(s)
	defer C.free(unsafe.Pointer(cs))
	C.gtk_label_set_text(w, cs)
}
func setEntry(w unsafe.Pointer, s string) {
	cs := cstr(s)
	defer C.free(unsafe.Pointer(cs))
	C.gtk_entry_set_text(w, cs)
}
func entryText(w unsafe.Pointer) string {
	p := C.gtk_entry_get_text(w)
	if p == nil {
		return ""
	}
	return C.GoString(p)
}
func setTextView(w unsafe.Pointer, s string) {
	b := C.gtk_text_view_get_buffer(w)
	cs := cstr(s)
	defer C.free(unsafe.Pointer(cs))
	C.gtk_text_buffer_set_text(unsafe.Pointer(b), cs, C.int(-1))
}
func appendLog(s string) {
	queueUI(func() {
		b := C.gtk_text_view_get_buffer(logView)
		cs := cstr(s + "\n")
		defer C.free(unsafe.Pointer(cs))
		C.gtk_text_buffer_insert_at_cursor(unsafe.Pointer(b), cs, C.int(-1))
	})
}
func queueUI(fn func()) { uiMu.Lock(); uiQueue = append(uiQueue, fn); uiMu.Unlock(); C.schedule_idle() }

//export goDrainUI
func goDrainUI() {
	uiMu.Lock()
	q := uiQueue
	uiQueue = nil
	uiMu.Unlock()
	for _, fn := range q {
		fn()
	}
}

//export goSearchChanged
func goSearchChanged() { applyFilter(entryText(searchEntry)) }

//export goClicked
func goClicked(id C.int) {
	switch int(id) {
	case idPrev:
		if page > 0 {
			page--
			renderRows()
		}
	case idNext:
		if (page+1)*visibleRows < len(filtered) {
			page++
			renderRows()
		}
	case idNextPage:
		collectVisible()
		buildSummary()
		showPage(2)
	case idBack:
		showPage(1)
	case idInstall:
		startOperation(false)
	case idRemove:
		startOperation(true)
	case idCancel:
		cancelInstall()
	case idFinish:
		buildFinish()
		showPage(3)
	case idClose:
		C.gtk_main_quit()
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--diagnose" {
		diagnose()
		return
	}
	distroName, packageManager = detectLinux()
	packageSet = loadPackageSet(packageManager)
	flatpakSet = loadFlatpakSet()
	apps = resolveLinuxApps()
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
	states = make([]appState, len(apps))
	operationResults = map[int]operationResult{}
	currentOperation = "install"
	for i := range states {
		states[i].Version = "Legújabb"
		states[i].Versions = []string{"Legújabb"}
	}
	if C.gtk_init_check(nil, nil) == 0 {
		fmt.Fprintln(os.Stderr, "GTK3 grafikus környezet nem elérhető.")
		os.Exit(2)
	}
	buildUI()
	applyFilter("")
	preloadVisibleVersions()
	C.gtk_main()
}

func diagnose() {
	d, p := detectLinux()
	ps := loadPackageSet(p)
	fs := loadFlatpakSet()
	distroName = d
	packageManager = p
	packageSet = ps
	flatpakSet = fs
	aa := resolveLinuxApps()
	fmt.Printf("OS=linux\nDistro=%s\nPackageManager=%s\nNativePackages=%d\nFlatpakApps=%d\nVisibleCatalog=%d\n", d, p, len(ps), len(fs), len(aa))
	for i, a := range aa {
		if i >= 30 {
			break
		}
		fmt.Printf("%s\t%s:%s\n", a.Name, a.Provider, a.Package)
	}
}

func detectLinux() (string, string) {
	name := "Linux"
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(ln, "PRETTY_NAME=") {
				name = strings.Trim(strings.TrimPrefix(ln, "PRETTY_NAME="), "\"")
			}
		}
	}
	for _, p := range []string{"apt-get", "dnf", "pacman", "zypper"} {
		if _, err := exec.LookPath(p); err == nil {
			return name, p
		}
	}
	return name, ""
}

func loadPackageSet(pm string) map[string]bool {
	out := map[string]bool{}
	var cmd *exec.Cmd
	switch pm {
	case "apt-get":
		cmd = exec.Command("apt-cache", "pkgnames")
	case "dnf":
		cmd = exec.Command("dnf", "repoquery", "--qf", "%{name}")
	case "pacman":
		cmd = exec.Command("pacman", "-Slq")
	case "zypper":
		cmd = exec.Command("zypper", "--non-interactive", "search", "-s", "-t", "package")
	default:
		return out
	}
	b, err := cmd.Output()
	if err != nil {
		return out
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" {
			continue
		}
		if pm == "zypper" {
			parts := strings.Split(ln, "|")
			if len(parts) >= 3 {
				n := strings.TrimSpace(parts[1])
				if n != "Name" && n != "" {
					out[n] = true
				}
			}
			continue
		}
		f := strings.Fields(ln)
		if len(f) > 0 {
			out[f[0]] = true
		}
	}
	return out
}

func loadFlatpakSet() map[string]string {
	out := map[string]string{}
	if _, err := exec.LookPath("flatpak"); err != nil {
		return out
	}
	b, err := exec.Command("flatpak", "remote-ls", "--columns=remote,application").Output()
	if err != nil {
		return out
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		parts := strings.Fields(ln)
		if len(parts) >= 2 {
			remote := parts[0]
			appID := parts[len(parts)-1]
			if remote != "" && appID != "" {
				out[appID] = remote
			}
		}
	}
	return out
}

func firstPresent(xs []string, set map[string]bool) (string, bool) {
	for _, x := range xs {
		if set[x] {
			return x, true
		}
	}
	return "", false
}
func resolveLinuxApps() []linuxApp {
	var out []linuxApp
	added := map[string]bool{}
	add := func(a linuxApp) {
		k := strings.ToLower(strings.TrimSpace(a.Name))
		if added[k] {
			return
		}
		added[k] = true
		out = append(out, a)
	}
	for _, c := range candidates {
		var ids []string
		switch packageManager {
		case "apt-get":
			ids = c.Apt
		case "dnf":
			ids = c.Dnf
		case "pacman":
			ids = c.Pacman
		case "zypper":
			ids = c.Zypper
		}
		if p, ok := firstPresent(ids, packageSet); ok {
			add(linuxApp{linuxCandidate: c, Provider: packageManager, Package: p, InstallPath: "Rendszer által kezelt (/usr, /opt, ... )"})
			continue
		}
		for _, p := range c.Flatpak {
			if remote, ok := flatpakSet[p]; ok {
				add(linuxApp{linuxCandidate: c, Provider: "flatpak", Package: p, FlatpakRemote: remote, InstallPath: "Felhasználói Flatpak (~/.local/share/flatpak)"})
				break
			}
		}
	}

	for _, g := range genericCatalog {
		if added[strings.ToLower(strings.TrimSpace(g.Name))] {
			continue
		}
		for _, form := range genericPackageForms(g.Name) {
			if packageSet[form] {
				add(linuxApp{linuxCandidate: linuxCandidate{Name: g.Name, Category: g.Category}, Provider: packageManager, Package: form, InstallPath: "Rendszer által kezelt (/usr, /opt, ... )"})
				break
			}
		}
	}
	return out
}

func genericPackageForms(name string) []string {
	n := strings.ToLower(strings.TrimSpace(name))
	repl := strings.NewReplacer("+", "plus", "/", "-", "_", "-", ".", "-", "(", "-", ")", "-", ":", "-", "&", "and", "'", "", "’", "")
	n = repl.Replace(n)
	var b strings.Builder
	lastDash := false
	for _, r := range n {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	hy := strings.Trim(b.String(), "-")
	compact := strings.ReplaceAll(hy, "-", "")
	out := []string{}
	seen := map[string]bool{}
	for _, v := range []string{hy, compact} {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func buildUI() {
	window = unsafe.Pointer(C.gtk_window_new(0))
	t := cstr(appTitle)
	C.gtk_window_set_title(window, t)
	C.free(unsafe.Pointer(t))
	C.gtk_window_set_default_size(window, 1225, 910)
	C.gtk_window_set_resizable(window, 0)
	C.connect_destroy(window)
	if exe, err := os.Executable(); err == nil {
		icon := filepath.Join(filepath.Dir(exe), "Wiz4rdFr0g_icon.png")
		if _, e := os.Stat(icon); e == nil {
			ci := cstr(icon)
			C.gtk_window_set_icon_from_file(window, ci, nil)
			C.free(unsafe.Pointer(ci))
		}
	}
	root := unsafe.Pointer(C.gtk_box_new(1, 8))
	C.gtk_container_set_border_width(root, 16)
	C.gtk_container_add(window, root)
	page1 = buildPage1()
	page2 = buildPage2()
	page3 = buildPage3()
	C.gtk_box_pack_start(root, page1, 1, 1, 0)
	C.gtk_box_pack_start(root, page2, 1, 1, 0)
	C.gtk_box_pack_start(root, page3, 1, 1, 0)
	C.gtk_widget_show_all(window)
	showPage(1)
}

func label(s string) unsafe.Pointer {
	cs := cstr(s)
	w := unsafe.Pointer(C.gtk_label_new(cs))
	C.free(unsafe.Pointer(cs))
	C.gtk_label_set_xalign(w, 0)
	return w
}
func button(s string, id int) unsafe.Pointer {
	cs := cstr(s)
	w := unsafe.Pointer(C.gtk_button_new_with_label(cs))
	C.free(unsafe.Pointer(cs))
	C.connect_clicked(w, C.int(id))
	return w
}
func buildPage1() unsafe.Pointer {
	box := unsafe.Pointer(C.gtk_box_new(1, 8))
	info := label(fmt.Sprintf("Linux: %s | csomagkezelő: %s | elérhető programok: %d", distroName, packageManager, len(apps)))
	C.gtk_box_pack_start(box, info, 0, 0, 0)
	searchEntry = unsafe.Pointer(C.gtk_entry_new())
	ph := cstr("Program keresése...")
	C.gtk_entry_set_placeholder_text(searchEntry, ph)
	C.free(unsafe.Pointer(ph))
	C.connect_changed(searchEntry)
	C.gtk_box_pack_start(box, searchEntry, 0, 0, 0)
	countLabel = label("")
	C.gtk_box_pack_start(box, countLabel, 0, 0, 0)
	grid := unsafe.Pointer(C.gtk_grid_new())
	C.gtk_grid_set_row_spacing(grid, 6)
	C.gtk_grid_set_column_spacing(grid, 10)
	headers := []string{"", "Program", "Verzió", "Telepítési hely"}
	for i, h := range headers {
		C.gtk_grid_attach(grid, label(h), C.int(i), 0, 1, 1)
	}
	rows = make([]*rowUI, visibleRows)
	for i := 0; i < visibleRows; i++ {
		ck := unsafe.Pointer(C.gtk_check_button_new())
		nm := label("")
		combo := unsafe.Pointer(C.gtk_combo_box_text_new())
		path := unsafe.Pointer(C.gtk_entry_new())
		C.gtk_editable_set_editable(path, 0)
		C.gtk_widget_set_size_request(nm, 330, -1)
		C.gtk_widget_set_size_request(combo, 260, -1)
		C.gtk_widget_set_size_request(path, 420, -1)
		C.gtk_grid_attach(grid, ck, 0, C.int(i+1), 1, 1)
		C.gtk_grid_attach(grid, nm, 1, C.int(i+1), 1, 1)
		C.gtk_grid_attach(grid, combo, 2, C.int(i+1), 1, 1)
		C.gtk_grid_attach(grid, path, 3, C.int(i+1), 1, 1)
		rows[i] = &rowUI{-1, ck, nm, combo, path}
	}
	sw := unsafe.Pointer(C.gtk_scrolled_window_new(nil, nil))
	C.gtk_scrolled_window_set_policy(sw, 1, 1)
	C.gtk_container_add(sw, grid)
	C.gtk_box_pack_start(box, sw, 1, 1, 0)
	nav := unsafe.Pointer(C.gtk_box_new(0, 8))
	prevBtn = button("Előző oldal", idPrev)
	nextBtn = button("Következő oldal", idNext)
	nextPageBtn = button("Tovább", idNextPage)
	C.gtk_box_pack_start(nav, prevBtn, 0, 0, 0)
	C.gtk_box_pack_start(nav, nextBtn, 0, 0, 0)
	C.gtk_box_pack_end(nav, nextPageBtn, 0, 0, 0)
	C.gtk_box_pack_end(box, nav, 0, 0, 0)
	return box
}
func buildPage2() unsafe.Pointer {
	box := unsafe.Pointer(C.gtk_box_new(1, 8))
	C.gtk_box_pack_start(box, label("Kiválasztott programok"), 0, 0, 0)
	summaryView = unsafe.Pointer(C.gtk_text_view_new())
	C.gtk_text_view_set_editable(summaryView, 0)
	C.gtk_text_view_set_cursor_visible(summaryView, 0)
	sw1 := unsafe.Pointer(C.gtk_scrolled_window_new(nil, nil))
	C.gtk_container_add(sw1, summaryView)
	C.gtk_widget_set_size_request(sw1, -1, 220)
	C.gtk_box_pack_start(box, sw1, 0, 1, 0)
	C.gtk_box_pack_start(box, label("Napló"), 0, 0, 0)
	logView = unsafe.Pointer(C.gtk_text_view_new())
	C.gtk_text_view_set_editable(logView, 0)
	C.gtk_text_view_set_cursor_visible(logView, 0)
	C.gtk_text_view_set_monospace(logView, 1)
	sw2 := unsafe.Pointer(C.gtk_scrolled_window_new(nil, nil))
	C.gtk_container_add(sw2, logView)
	C.gtk_box_pack_start(box, sw2, 1, 1, 0)
	nav := unsafe.Pointer(C.gtk_box_new(0, 8))
	backBtn = button("Vissza", idBack)
	installBtn = button("Telepít", idInstall)
	removeBtn = button("Eltávolít", idRemove)
	cancelBtn = button("Megszakítás", idCancel)
	finishBtn = button("Tovább", idFinish)
	C.gtk_box_pack_start(nav, backBtn, 0, 0, 0)
	C.gtk_box_pack_start(nav, installBtn, 0, 0, 0)
	C.gtk_box_pack_start(nav, removeBtn, 0, 0, 0)
	C.gtk_box_pack_start(nav, cancelBtn, 0, 0, 0)
	C.gtk_box_pack_end(nav, finishBtn, 0, 0, 0)
	C.gtk_box_pack_end(box, nav, 0, 0, 0)
	return box
}
func buildPage3() unsafe.Pointer {
	box := unsafe.Pointer(C.gtk_box_new(1, 8))
	C.gtk_box_pack_start(box, label("Befejezés"), 0, 0, 0)
	finishView = unsafe.Pointer(C.gtk_text_view_new())
	C.gtk_text_view_set_editable(finishView, 0)
	C.gtk_text_view_set_cursor_visible(finishView, 0)
	sw := unsafe.Pointer(C.gtk_scrolled_window_new(nil, nil))
	C.gtk_container_add(sw, finishView)
	C.gtk_box_pack_start(box, sw, 1, 1, 0)
	closeBtn = button("Bezárás", idClose)
	C.gtk_box_pack_end(box, closeBtn, 0, 0, 0)
	return box
}
func showPage(n int) {
	current := []unsafe.Pointer{page1, page2, page3}
	for i, w := range current {
		if i == n-1 {
			C.gtk_widget_show(w)
		} else {
			C.gtk_widget_hide(w)
		}
	}
}

func applyFilter(q string) {
	collectVisible()
	q = strings.ToLower(strings.TrimSpace(q))
	filtered = nil
	for i, a := range apps {
		if q == "" || strings.Contains(strings.ToLower(a.Name), q) || strings.Contains(strings.ToLower(a.Category), q) {
			filtered = append(filtered, i)
		}
	}
	page = 0
	renderRows()
}
func renderRows() {
	start := page * visibleRows
	for i, r := range rows {
		idx := start + i
		if idx >= len(filtered) {
			r.catalogIndex = -1
			setLabel(r.name, "")
			C.gtk_combo_box_text_remove_all(r.combo)
			setEntry(r.path, "")
			C.gtk_toggle_button_set_active(r.check, 0)
			C.gtk_widget_set_sensitive(r.check, 0)
			C.gtk_widget_set_sensitive(r.combo, 0)
			continue
		}
		ci := filtered[idx]
		r.catalogIndex = ci
		a := apps[ci]
		st := &states[ci]
		setLabel(r.name, a.Name+"  ["+a.Category+"]")
		C.gtk_widget_set_sensitive(r.check, 1)
		C.gtk_widget_set_sensitive(r.combo, 1)
		C.gtk_toggle_button_set_active(r.check, C.gboolean(boolInt(st.Selected)))
		setEntry(r.path, a.InstallPath)
		fillCombo(r.combo, st.Versions, st.Version)
	}
	setLabel(countLabel, fmt.Sprintf("Találatok: %d | oldal: %d / %d", len(filtered), page+1, max(1, (len(filtered)+visibleRows-1)/visibleRows)))
	C.gtk_widget_set_sensitive(prevBtn, C.gboolean(boolInt(page > 0)))
	C.gtk_widget_set_sensitive(nextBtn, C.gboolean(boolInt((page+1)*visibleRows < len(filtered))))
	preloadVisibleVersions()
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func fillCombo(w unsafe.Pointer, versions []string, selected string) {
	C.gtk_combo_box_text_remove_all(w)
	sel := 0
	for i, v := range versions {
		cs := cstr(v)
		C.gtk_combo_box_text_append_text(w, cs)
		C.free(unsafe.Pointer(cs))
		if v == selected {
			sel = i
		}
	}
	C.gtk_combo_box_set_active(w, C.int(sel))
}
func comboText(w unsafe.Pointer) string {
	p := C.gtk_combo_box_text_get_active_text(w)
	if p == nil {
		return "Legújabb"
	}
	defer C.g_free(unsafe.Pointer(p))
	return C.GoString(p)
}
func collectVisible() {
	for _, r := range rows {
		if r.catalogIndex < 0 {
			continue
		}
		st := &states[r.catalogIndex]
		st.Selected = C.gtk_toggle_button_get_active(r.check) != 0
		st.Version = comboText(r.combo)
	}
}

func preloadVisibleVersions() {
	for _, r := range rows {
		if r.catalogIndex < 0 {
			continue
		}
		idx := r.catalogIndex
		if states[idx].Loading || len(states[idx].Versions) > 1 {
			continue
		}
		states[idx].Loading = true
		go func(i int) {
			vs := loadVersions(apps[i])
			queueUI(func() {
				states[i].Loading = false
				if len(vs) > 0 {
					states[i].Versions = append([]string{"Legújabb"}, vs...)
				}
				renderRows()
			})
		}(idx)
	}
}
func loadVersions(a linuxApp) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	switch a.Provider {
	case "apt-get":
		cmd = exec.CommandContext(ctx, "apt-cache", "madison", a.Package)
	case "dnf":
		cmd = exec.CommandContext(ctx, "dnf", "--showduplicates", "list", a.Package, "-q")
	case "pacman":

		return nil
	case "zypper":
		cmd = exec.CommandContext(ctx, "zypper", "--non-interactive", "search", "-s", "-x", a.Package)
	default:
		return nil
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	set := map[string]bool{}
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		var v string
		switch a.Provider {
		case "apt-get":
			p := strings.Split(ln, "|")
			if len(p) >= 2 {
				v = strings.TrimSpace(p[1])
			}
		case "dnf":
			f := strings.Fields(ln)
			if len(f) >= 2 && !strings.HasPrefix(f[0], "Installed") {
				v = f[1]
			}
		case "pacman":
			if strings.HasPrefix(strings.TrimSpace(ln), "Version") {
				p := strings.SplitN(ln, ":", 2)
				if len(p) == 2 {
					v = strings.TrimSpace(p[1])
				}
			}
		case "zypper":
			p := strings.Split(ln, "|")
			if len(p) >= 4 {
				v = strings.TrimSpace(p[3])
			}
		}
		if v != "" && !set[v] {
			set[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func buildSummary() {
	collectVisible()
	var b strings.Builder
	count := 0
	for i, a := range apps {
		if states[i].Selected {
			count++
			fmt.Fprintf(&b, "%s\n  Verzió: %s\n  Forrás: %s (%s)\n  Telepítési hely: %s\n\n", a.Name, states[i].Version, a.Provider, a.Package, a.InstallPath)
		}
	}
	if count == 0 {
		b.WriteString("Nincs kiválasztott program.\n")
	}
	setTextView(summaryView, b.String())
	setTextView(logView, fmt.Sprintf("[INFO] OS: Linux - %s\n[INFO] Csomagkezelő: %s\n", distroName, packageManager))
	C.gtk_widget_set_sensitive(installBtn, C.gboolean(boolInt(count > 0)))
	C.gtk_widget_set_sensitive(removeBtn, C.gboolean(boolInt(count > 0)))
	C.gtk_widget_set_sensitive(finishBtn, 0)
}

func startOperation(remove bool) {
	opMu.Lock()
	if opCancel != nil {
		opMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	opCancel = cancel
	opMu.Unlock()
	operationResults = map[int]operationResult{}
	if remove {
		currentOperation = "remove"
	} else {
		currentOperation = "install"
	}
	C.gtk_widget_set_sensitive(installBtn, 0)
	C.gtk_widget_set_sensitive(removeBtn, 0)
	C.gtk_widget_set_sensitive(backBtn, 0)
	C.gtk_widget_set_sensitive(finishBtn, 0)
	if remove {
		appendLog("[INFO] Eltávolítás indítása...")
	} else {
		appendLog("[INFO] Telepítés indítása...")
	}
	go func() {
		for i, a := range apps {
			if !states[i].Selected {
				continue
			}
			appendLog(fmt.Sprintf("[INFO] %s", a.Name))
			if remove {
				installed, err := isInstalledExact(ctx, a)
				if err != nil {
					operationResults[i] = operationResult{false, "Telepítettség ellenőrzése sikertelen: " + err.Error()}
					appendLog("[ERROR] Telepítettség ellenőrzése sikertelen: " + err.Error())
					continue
				}
				if !installed {
					operationResults[i] = operationResult{false, "A pontos csomagazonosító nincs telepítve; eltávolítás nem indult"}
					appendLog("[WARN] A pontos csomagazonosító nincs telepítve; eltávolítás nem indult")
					continue
				}
			}

			spec, err := operationSpec(a, states[i].Version, remove)
			if err != nil {
				operationResults[i] = operationResult{false, err.Error()}
				appendLog("[ERROR] " + err.Error())
				continue
			}
			args, cmdName := executableSpec(spec)
			if cmdName == "" {
				operationResults[i] = operationResult{false, "Nincs használható jogosultság-emelési mód"}
				appendLog("[ERROR] Nincs használható jogosultság-emelési mód")
				continue
			}
			appendLog("[CMD] " + cmdName + " " + strings.Join(args, " "))
			cmd := exec.CommandContext(ctx, cmdName, args...)
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()
			if err := cmd.Start(); err != nil {
				operationResults[i] = operationResult{false, err.Error()}
				appendLog("[ERROR] " + err.Error())
				continue
			}
			var wg sync.WaitGroup
			wg.Add(2)
			scan := func(r *bufio.Scanner, tag string) {
				defer wg.Done()
				for r.Scan() {
					appendLog(tag + " " + r.Text())
				}
			}
			go scan(bufio.NewScanner(stdout), "[OUT]")
			go scan(bufio.NewScanner(stderr), "[STDERR]")
			err = cmd.Wait()
			wg.Wait()
			if err != nil {
				operationResults[i] = operationResult{false, err.Error()}
				appendLog("[ERROR] " + a.Name + ": " + err.Error())
				continue
			}
			wantInstalled := !remove
			if err := verifyInstalledState(ctx, a, wantInstalled); err != nil {
				operationResults[i] = operationResult{false, err.Error()}
				appendLog("[ERROR] " + a.Name + ": " + err.Error())
				continue
			}
			operationResults[i] = operationResult{true, ""}
			if remove {
				appendLog("[SUCCESS] " + a.Name + " eltávolítva és az eltűnés ellenőrizve")
			} else {
				appendLog("[SUCCESS] " + a.Name + " telepítve és jelenlét ellenőrizve")
			}
		}
		opMu.Lock()
		opCancel = nil
		opMu.Unlock()
		queueUI(func() {
			C.gtk_widget_set_sensitive(backBtn, 1)
			C.gtk_widget_set_sensitive(finishBtn, 1)
			if remove {
				appendLog("[INFO] Eltávolítási folyamat véget ért.")
			} else {
				appendLog("[INFO] Telepítési folyamat véget ért.")
			}
		})
	}()
}
func privilegedCommand(args ...string) ([]string, string) {
	if len(args) == 0 {
		return nil, ""
	}
	if os.Geteuid() == 0 {
		return args[1:], args[0]
	}
	if _, err := exec.LookPath("pkexec"); err == nil {
		return args, "pkexec"
	}
	if _, err := exec.LookPath("sudo"); err == nil {
		return args, "sudo"
	}
	return nil, ""
}

func operationSpec(a linuxApp, version string, remove bool) (linuxpkg.Spec, error) {
	if remove {
		return linuxpkg.Remove(a.Provider, a.Package, a.FlatpakRemote)
	}
	return linuxpkg.Install(a.Provider, a.Package, a.FlatpakRemote, version)
}

func executableSpec(spec linuxpkg.Spec) ([]string, string) {
	if !spec.NeedsRoot {
		return spec.Args, spec.Name
	}
	all := append([]string{spec.Name}, spec.Args...)
	return privilegedCommand(all...)
}

func isInstalledExact(ctx context.Context, a linuxApp) (bool, error) {
	spec, err := linuxpkg.Detect(a.Provider, a.Package, a.FlatpakRemote)
	if err != nil {
		return false, err
	}
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return linuxpkg.DetectionSuccess(a.Provider, out, true), nil
}

func verifyInstalledState(ctx context.Context, a linuxApp, wantInstalled bool) error {
	deadline := time.Now().Add(12 * time.Second)
	for {
		installed, err := isInstalledExact(ctx, a)
		if err != nil {
			return err
		}
		if installed == wantInstalled {
			return nil
		}
		if time.Now().After(deadline) {
			if wantInstalled {
				return fmt.Errorf("a telepítő sikeresen kilépett, de a pontos csomagazonosító nem észlelhető")
			}
			return fmt.Errorf("az eltávolító sikeresen kilépett, de a pontos csomagazonosító továbbra is telepítve van")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func cancelInstall() {
	opMu.Lock()
	c := opCancel
	opMu.Unlock()
	if c != nil {
		c()
		appendLog("[WARN] Megszakítás kérve.")
	}
}
func buildFinish() {
	var b strings.Builder
	ok, fail := 0, 0
	for i, a := range apps {
		if !states[i].Selected {
			continue
		}
		r, has := operationResults[i]
		if has && r.OK {
			ok++
			fmt.Fprintf(&b, "✔ %s\n", a.Name)
		} else {
			fail++
			err := "nem futott"
			if has && r.Err != "" {
				err = r.Err
			}
			fmt.Fprintf(&b, "✖ %s - %s\n", a.Name, err)
		}
	}
	operationName := "Telepítés"
	if currentOperation == "remove" {
		operationName = "Eltávolítás"
	}
	fmt.Fprintf(&b, "\n%s - sikeres: %d\n%s - sikertelen: %d\n", operationName, ok, operationName, fail)
	setTextView(finishView, b.String())
}

var genericCatalog = []struct{ Name, Category string }{
	{"Google Chrome", "BÖNGÉSZŐK"},
	{"Mozilla Firefox", "BÖNGÉSZŐK"},
	{"Microsoft Edge", "BÖNGÉSZŐK"},
	{"Brave", "BÖNGÉSZŐK"},
	{"Opera", "BÖNGÉSZŐK"},
	{"Vivaldi", "BÖNGÉSZŐK"},
	{"Zoom", "KOMMUNIKÁCIÓ"},
	{"Discord", "KOMMUNIKÁCIÓ"},
	{"Microsoft Teams", "KOMMUNIKÁCIÓ"},
	{"Pidgin", "KOMMUNIKÁCIÓ"},
	{"Mozilla Thunderbird", "KOMMUNIKÁCIÓ"},
	{"Trillian", "KOMMUNIKÁCIÓ"},
	{"Telegram Desktop", "KOMMUNIKÁCIÓ"},
	{"iTunes", "MÉDIA / ZENE / VIDEÓ"},
	{"VLC Media Player", "MÉDIA / ZENE / VIDEÓ"},
	{"AIMP", "MÉDIA / ZENE / VIDEÓ"},
	{"foobar2000", "MÉDIA / ZENE / VIDEÓ"},
	{"Winamp", "MÉDIA / ZENE / VIDEÓ"},
	{"MusicBee", "MÉDIA / ZENE / VIDEÓ"},
	{"Audacity", "MÉDIA / ZENE / VIDEÓ"},
	{"K-Lite Codec Pack", "MÉDIA / ZENE / VIDEÓ"},
	{"GOM Player", "MÉDIA / ZENE / VIDEÓ"},
	{"Spotify", "MÉDIA / ZENE / VIDEÓ"},
	{"Combined Community Codec Pack (CCCP)", "MÉDIA / ZENE / VIDEÓ"},
	{"MediaMonkey", "MÉDIA / ZENE / VIDEÓ"},
	{"HandBrake", "MÉDIA / ZENE / VIDEÓ"},
	{"Krita", "GRAFIKA / KÉP / VIDEÓ"},
	{"Blender", "GRAFIKA / KÉP / VIDEÓ"},
	{"Paint.NET", "GRAFIKA / KÉP / VIDEÓ"},
	{"GIMP", "GRAFIKA / KÉP / VIDEÓ"},
	{"IrfanView", "GRAFIKA / KÉP / VIDEÓ"},
	{"XnView", "GRAFIKA / KÉP / VIDEÓ"},
	{"Inkscape", "GRAFIKA / KÉP / VIDEÓ"},
	{"FastStone Image Viewer", "GRAFIKA / KÉP / VIDEÓ"},
	{"Greenshot", "GRAFIKA / KÉP / VIDEÓ"},
	{"ShareX", "GRAFIKA / KÉP / VIDEÓ"},
	{"OBS Studio", "GRAFIKA / KÉP / VIDEÓ"},
	{"Foxit Reader", "DOKUMENTUMOK / PDF / IRODA"},
	{"LibreOffice", "DOKUMENTUMOK / PDF / IRODA"},
	{"SumatraPDF", "DOKUMENTUMOK / PDF / IRODA"},
	{"CutePDF", "DOKUMENTUMOK / PDF / IRODA"},
	{"OpenOffice", "DOKUMENTUMOK / PDF / IRODA"},
	{"Adobe Acrobat Reader", "DOKUMENTUMOK / PDF / IRODA"},
	{"Malwarebytes", "BIZTONSÁG"},
	{"Avast", "BIZTONSÁG"},
	{"AVG", "BIZTONSÁG"},
	{"Spybot 2", "BIZTONSÁG"},
	{"Avira", "BIZTONSÁG"},
	{"SUPERAntiSpyware", "BIZTONSÁG"},
	{"qBittorrent", "FÁJLMEGOSZTÁS / TORRENT"},
	{"Dropbox", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Google Drive for Desktop", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Microsoft OneDrive", "FELHŐ / SZINKRONIZÁLÁS"},
	{"SugarSync", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Steam", "JÁTÉK / ÁLTALÁNOS"},
	{"Epic Games Launcher", "JÁTÉK / ÁLTALÁNOS"},
	{"Evernote", "JÁTÉK / ÁLTALÁNOS"},
	{"Google Earth", "JÁTÉK / ÁLTALÁNOS"},
	{"KeePass 2", "JÁTÉK / ÁLTALÁNOS"},
	{"Everything", "JÁTÉK / ÁLTALÁNOS"},
	{"NV Access", "JÁTÉK / ÁLTALÁNOS"},
	{"AnyDesk", "SEGÉDPROGRAMOK"},
	{"TeamViewer 15", "SEGÉDPROGRAMOK"},
	{"ImgBurn", "SEGÉDPROGRAMOK"},
	{"RealVNC Server", "SEGÉDPROGRAMOK"},
	{"RealVNC Viewer", "SEGÉDPROGRAMOK"},
	{"TightVNC", "SEGÉDPROGRAMOK"},
	{"TeraCopy", "SEGÉDPROGRAMOK"},
	{"CDBurnerXP", "SEGÉDPROGRAMOK"},
	{"Revo Uninstaller", "SEGÉDPROGRAMOK"},
	{"Launchy", "SEGÉDPROGRAMOK"},
	{"WinDirStat", "SEGÉDPROGRAMOK"},
	{"WizTree", "SEGÉDPROGRAMOK"},
	{"Glary Utilities", "SEGÉDPROGRAMOK"},
	{"InfraRecorder", "SEGÉDPROGRAMOK"},
	{"Open-Shell", "SEGÉDPROGRAMOK"},
	{"CCleaner", "SEGÉDPROGRAMOK"},
	{"Microsoft PowerToys", "SEGÉDPROGRAMOK"},
	{"Rufus", "SEGÉDPROGRAMOK"},
	{"balenaEtcher", "SEGÉDPROGRAMOK"},
	{"TreeSize Free", "SEGÉDPROGRAMOK"},
	{"Windows Terminal", "SEGÉDPROGRAMOK"},
	{"7-Zip", "TÖMÖRÍTÉS"},
	{"PeaZip", "TÖMÖRÍTÉS"},
	{"WinRAR", "TÖMÖRÍTÉS"},
	{"Python 3", "FEJLESZTŐI ESZKÖZÖK"},
	{"Python 2 / Legacy Python", "FEJLESZTŐI ESZKÖZÖK"},
	{"Git", "FEJLESZTŐI ESZKÖZÖK"},
	{"GitHub Desktop", "FEJLESZTŐI ESZKÖZÖK"},
	{"Node.js", "FEJLESZTŐI ESZKÖZÖK"},
	{"FileZilla", "FEJLESZTŐI ESZKÖZÖK"},
	{"Notepad++", "FEJLESZTŐI ESZKÖZÖK"},
	{"WinSCP", "FEJLESZTŐI ESZKÖZÖK"},
	{"PuTTY", "FEJLESZTŐI ESZKÖZÖK"},
	{"WinMerge", "FEJLESZTŐI ESZKÖZÖK"},
	{"Eclipse", "FEJLESZTŐI ESZKÖZÖK"},
	{"Visual Studio Code", "FEJLESZTŐI ESZKÖZÖK"},
	{"Cursor", "FEJLESZTŐI ESZKÖZÖK"},
	{"Docker Desktop", "FEJLESZTŐI ESZKÖZÖK"},
	{"Postman", "FEJLESZTŐI ESZKÖZÖK"},
	{"DBeaver", "FEJLESZTŐI ESZKÖZÖK"},
	{"MySQL Workbench", "FEJLESZTŐI ESZKÖZÖK"},
	{"Visual Studio Community", "FEJLESZTŐI ESZKÖZÖK"},
	{"IntelliJ IDEA Community", "FEJLESZTŐI ESZKÖZÖK"},
	{"Android Studio", "FEJLESZTŐI ESZKÖZÖK"},
	{"GitKraken", "FEJLESZTŐI ESZKÖZÖK"},
	{"Adoptium / AdoptOpenJDK JDK 8", "JAVA / JDK / JRE"},
	{"Adoptium / AdoptOpenJDK JDK 11", "JAVA / JDK / JRE"},
	{"Adoptium / AdoptOpenJDK JDK 17", "JAVA / JDK / JRE"},
	{"Adoptium / AdoptOpenJDK JDK 21", "JAVA / JDK / JRE"},
	{"Adoptium / AdoptOpenJDK JDK 25", "JAVA / JDK / JRE"},
	{"Amazon Corretto JDK 8", "JAVA / JDK / JRE"},
	{"Amazon Corretto JDK 11", "JAVA / JDK / JRE"},
	{"Amazon Corretto JDK 17", "JAVA / JDK / JRE"},
	{"Amazon Corretto JDK 21", "JAVA / JDK / JRE"},
	{"Amazon Corretto JDK 25", "JAVA / JDK / JRE"},
	{"Amazon Corretto JRE 8", "JAVA / JDK / JRE"},
	{".NET Framework 4.8.1", ".NET RUNTIME-OK"},
	{".NET Desktop Runtime 8", ".NET RUNTIME-OK"},
	{".NET Desktop Runtime 9", ".NET RUNTIME-OK"},
	{".NET Desktop Runtime 10", ".NET RUNTIME-OK"},
	{"ASP.NET Core Runtime 8", ".NET RUNTIME-OK"},
	{"ASP.NET Core Runtime 9", ".NET RUNTIME-OK"},
	{"ASP.NET Core Runtime 10", ".NET RUNTIME-OK"},
	{"Microsoft Visual C++ Redistributable 2015+", "MICROSOFT VISUAL C++ REDISTRIBUTABLE"},
	{"Microsoft Visual C++ Redistributable 2013", "MICROSOFT VISUAL C++ REDISTRIBUTABLE"},
	{"Microsoft Visual C++ Redistributable 2012", "MICROSOFT VISUAL C++ REDISTRIBUTABLE"},
	{"Microsoft Visual C++ Redistributable 2010", "MICROSOFT VISUAL C++ REDISTRIBUTABLE"},
	{"Microsoft Visual C++ Redistributable 2008", "MICROSOFT VISUAL C++ REDISTRIBUTABLE"},
	{"Microsoft Visual C++ Redistributable 2005", "MICROSOFT VISUAL C++ REDISTRIBUTABLE"},
	{"CPU-Z", "RENDSZERINFORMÁCIÓ / HARDVER / TESZT"},
	{"HWiNFO", "RENDSZERINFORMÁCIÓ / HARDVER / TESZT"},
	{"CrystalDiskInfo", "RENDSZERINFORMÁCIÓ / HARDVER / TESZT"},
	{"CrystalDiskMark", "RENDSZERINFORMÁCIÓ / HARDVER / TESZT"},
	{"FanControl", "RENDSZERINFORMÁCIÓ / HARDVER / TESZT"},
	{"MSI Afterburner", "RENDSZERINFORMÁCIÓ / HARDVER / TESZT"},
	{"iLok License Manager", "AUDIO / ZENE / HANGSZER"},
	{"Focusrite Control 2", "AUDIO / ZENE / HANGSZER"},
	{"Softube Central", "AUDIO / ZENE / HANGSZER"},
	{"Amped Roots Free", "AUDIO / ZENE / HANGSZER"},
	{"Neural Amp Modeler", "AUDIO / ZENE / HANGSZER"},
	{"Tonocracy", "AUDIO / ZENE / HANGSZER"},
	{"TDR Nova", "AUDIO / ZENE / HANGSZER"},
	{"Vital", "AUDIO / ZENE / HANGSZER"},
	{"Spitfire LABS", "AUDIO / ZENE / HANGSZER"},
	{"ASIO4ALL", "AUDIO / ZENE / HANGSZER"},
	{"REAPER", "AUDIO / ZENE / HANGSZER"},
	{"FL Studio", "AUDIO / ZENE / HANGSZER"},
	{"Cubase", "AUDIO / ZENE / HANGSZER"},
	{"Archetype: Gojira X", "AUDIO / ZENE / HANGSZER"},
	{"Unity Hub", "GAME DEVELOPMENT / 3D"},
	{"Unity Editor / Unity Personal", "GAME DEVELOPMENT / 3D"},
	{"Unreal Engine", "GAME DEVELOPMENT / 3D"},
	{"Godot Engine", "GAME DEVELOPMENT / 3D"},
	{"CMake", "FEJLESZTÉS"},
	{"Cisco Packet Tracer", "HÁLÓZAT / IT / LABOR"},
	{"Wireshark", "HÁLÓZAT / IT / LABOR"},
	{"Nmap", "HÁLÓZAT / IT / LABOR"},
	{"VirtualBox", "HÁLÓZAT / IT / LABOR"},
	{"VMware Workstation", "HÁLÓZAT / IT / LABOR"},
	{"Ventoy", "HÁLÓZAT / IT / LABOR"},
	{"Pentablet driver / kezelőszoftver", "HARDVER / PERIFÉRIA"},
	{"Microsoft Word", "MICROSOFT OFFICE / FIZETŐS ALKALMAZÁSOK"},
	{"Microsoft Excel", "MICROSOFT OFFICE / FIZETŐS ALKALMAZÁSOK"},
	{"Microsoft PowerPoint", "MICROSOFT OFFICE / FIZETŐS ALKALMAZÁSOK"},
	{"Microsoft Access", "MICROSOFT OFFICE / FIZETŐS ALKALMAZÁSOK"},
	{"GOG GALAXY", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"EA app", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Ubisoft Connect", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Battle.net", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Riot Client", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Rockstar Games Launcher", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"itch.io App", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Amazon Games App", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Xbox App", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Minecraft Launcher", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Prism Launcher", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"MultiMC", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"CurseForge", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Modrinth App", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Nexus Mods / Vortex", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Overwolf", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Playnite", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Heroic Games Launcher", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Lutris", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"GeForce NOW", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Moonlight", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Sunshine", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Parsec", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"Antstream Arcade", "JÁTÉKPLATFORMOK / LAUNCHEREK"},
	{"LogMeIn Hamachi", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Tailscale", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"ZeroTier", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"WireGuard", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"OpenVPN Connect", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"SoftEther VPN Client", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Proton VPN", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Mullvad VPN", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Windscribe", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Cloudflare WARP", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"TunnelBear", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"NordVPN", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Surfshark", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Private Internet Access", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"NetBird", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Headscale", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Nebula", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"Radmin VPN", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"GameRanger", "VIRTUÁLIS HÁLÓZAT / VPN / LAN"},
	{"TeamViewer", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"RustDesk", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"UltraVNC", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"Chrome Remote Desktop", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"Microsoft Remote Desktop", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"mRemoteNG", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"MobaXterm", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"Termius", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"NoMachine", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"Splashtop", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"DWService", "TÁVOLI ELÉRÉS / REMOTE DESKTOP"},
	{"Ollama", "LOKÁLIS AI / LLM / CHAT"},
	{"LM Studio", "LOKÁLIS AI / LLM / CHAT"},
	{"Jan", "LOKÁLIS AI / LLM / CHAT"},
	{"GPT4All", "LOKÁLIS AI / LLM / CHAT"},
	{"AnythingLLM", "LOKÁLIS AI / LLM / CHAT"},
	{"Open WebUI", "LOKÁLIS AI / LLM / CHAT"},
	{"llama.cpp", "LOKÁLIS AI / LLM / CHAT"},
	{"KoboldCpp", "LOKÁLIS AI / LLM / CHAT"},
	{"text-generation-webui", "LOKÁLIS AI / LLM / CHAT"},
	{"LocalAI", "LOKÁLIS AI / LLM / CHAT"},
	{"vLLM", "LOKÁLIS AI / LLM / CHAT"},
	{"OpenLLM", "LOKÁLIS AI / LLM / CHAT"},
	{"Pinokio", "LOKÁLIS AI / LLM / CHAT"},
	{"Open Interpreter", "LOKÁLIS AI / LLM / CHAT"},
	{"Continue", "LOKÁLIS AI / LLM / CHAT"},
	{"Aider", "LOKÁLIS AI / LLM / CHAT"},
	{"Cline", "LOKÁLIS AI / LLM / CHAT"},
	{"Roo Code", "LOKÁLIS AI / LLM / CHAT"},
	{"Claude Code", "LOKÁLIS AI / LLM / CHAT"},
	{"Codex CLI", "LOKÁLIS AI / LLM / CHAT"},
	{"OpenHands", "LOKÁLIS AI / LLM / CHAT"},
	{"Dify", "LOKÁLIS AI / LLM / CHAT"},
	{"Flowise", "LOKÁLIS AI / LLM / CHAT"},
	{"Langflow", "LOKÁLIS AI / LLM / CHAT"},
	{"AnythingLLM Desktop", "LOKÁLIS AI / LLM / CHAT"},
	{"PrivateGPT", "LOKÁLIS AI / LLM / CHAT"},
	{"Chatbox", "LOKÁLIS AI / LLM / CHAT"},
	{"Msty", "LOKÁLIS AI / LLM / CHAT"},
	{"Cherry Studio", "LOKÁLIS AI / LLM / CHAT"},
	{"Perplexica", "LOKÁLIS AI / LLM / CHAT"},
	{"Khoj", "LOKÁLIS AI / LLM / CHAT"},
	{"LibreChat", "LOKÁLIS AI / LLM / CHAT"},
	{"SillyTavern", "LOKÁLIS AI / LLM / CHAT"},
	{"ComfyUI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"AUTOMATIC1111 Stable Diffusion WebUI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Forge WebUI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Fooocus", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"InvokeAI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"SwarmUI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"DiffusionBee", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Stable Diffusion WebUI Forge", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Easy Diffusion", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Krita AI Diffusion plugin", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Draw Things", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Upscayl", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Real-ESRGAN", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Topaz Photo AI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Topaz Gigapixel AI", "AI KÉPGENERÁLÁS / STABLE DIFFUSION"},
	{"Windsurf", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"GitHub Copilot", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Tabby", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Sourcegraph Cody", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"JetBrains AI Assistant", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Amazon Q Developer", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Pieces for Developers", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Codeium / Windsurf plugin", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Continue.dev", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"Tabnine", "AI FEJLESZTŐI ESZKÖZÖK"},
	{"XAMPP", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"WampServer", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Laragon", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"MAMP", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"LocalWP", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"DDEV", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Devilbox", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Lando", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Podman Desktop", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Rancher Desktop", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"WSL", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"WSL2", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Apache HTTP Server", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Nginx", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Caddy", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"IIS Express", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"PHP", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Composer", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"npm", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"pnpm", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Yarn", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Bun", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Deno", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Python", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"pip", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Ruby", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Ruby on Rails", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Go", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Rust", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"Java JDK", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{".NET SDK", "WEBFEJLESZTŐI STACK / LOKÁLIS SZERVER"},
	{"MySQL", "ADATBÁZIS SZERVEREK"},
	{"MariaDB", "ADATBÁZIS SZERVEREK"},
	{"PostgreSQL", "ADATBÁZIS SZERVEREK"},
	{"SQLite", "ADATBÁZIS SZERVEREK"},
	{"MongoDB Community Server", "ADATBÁZIS SZERVEREK"},
	{"Redis", "ADATBÁZIS SZERVEREK"},
	{"Memcached", "ADATBÁZIS SZERVEREK"},
	{"Microsoft SQL Server Express", "ADATBÁZIS SZERVEREK"},
	{"Microsoft SQL Server Developer", "ADATBÁZIS SZERVEREK"},
	{"Oracle Database Free", "ADATBÁZIS SZERVEREK"},
	{"CockroachDB", "ADATBÁZIS SZERVEREK"},
	{"CouchDB", "ADATBÁZIS SZERVEREK"},
	{"Neo4j Community", "ADATBÁZIS SZERVEREK"},
	{"InfluxDB", "ADATBÁZIS SZERVEREK"},
	{"TimescaleDB", "ADATBÁZIS SZERVEREK"},
	{"ClickHouse", "ADATBÁZIS SZERVEREK"},
	{"DuckDB", "ADATBÁZIS SZERVEREK"},
	{"HeidiSQL", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"pgAdmin", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"SQL Server Management Studio", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"Azure Data Studio", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"SQLiteStudio", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"DB Browser for SQLite", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"MongoDB Compass", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"RedisInsight", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"JetBrains DataGrip", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"TablePlus", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"Beekeeper Studio", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"Navicat", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"DbGate", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"RazorSQL", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"SQuirreL SQL", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"Adminer", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"phpMyAdmin", "ADATBÁZIS KEZELŐK / SQL GUI"},
	{"Insomnia", "API / BACKEND / TESZTELÉS"},
	{"Bruno", "API / BACKEND / TESZTELÉS"},
	{"Hoppscotch", "API / BACKEND / TESZTELÉS"},
	{"HTTPie", "API / BACKEND / TESZTELÉS"},
	{"curl", "API / BACKEND / TESZTELÉS"},
	{"wget", "API / BACKEND / TESZTELÉS"},
	{"SoapUI", "API / BACKEND / TESZTELÉS"},
	{"Fiddler Classic", "API / BACKEND / TESZTELÉS"},
	{"Fiddler Everywhere", "API / BACKEND / TESZTELÉS"},
	{"Charles Proxy", "API / BACKEND / TESZTELÉS"},
	{"mitmproxy", "API / BACKEND / TESZTELÉS"},
	{"Burp Suite Community Edition", "API / BACKEND / TESZTELÉS"},
	{"OWASP ZAP", "API / BACKEND / TESZTELÉS"},
	{"k6", "API / BACKEND / TESZTELÉS"},
	{"Apache JMeter", "API / BACKEND / TESZTELÉS"},
	{"Locust", "API / BACKEND / TESZTELÉS"},
	{"Swagger Editor", "API / BACKEND / TESZTELÉS"},
	{"Swagger UI", "API / BACKEND / TESZTELÉS"},
	{"Stoplight Studio", "API / BACKEND / TESZTELÉS"},
	{"Yaak", "API / BACKEND / TESZTELÉS"},
	{"Paw / RapidAPI Client", "API / BACKEND / TESZTELÉS"},
	{"VSCodium", "IDE-K / KÓDSZERKESZTŐK"},
	{"Visual Studio Professional", "IDE-K / KÓDSZERKESZTŐK"},
	{"Visual Studio Enterprise", "IDE-K / KÓDSZERKESZTŐK"},
	{"Sublime Text", "IDE-K / KÓDSZERKESZTŐK"},
	{"Zed", "IDE-K / KÓDSZERKESZTŐK"},
	{"Eclipse IDE", "IDE-K / KÓDSZERKESZTŐK"},
	{"Apache NetBeans", "IDE-K / KÓDSZERKESZTŐK"},
	{"Code::Blocks", "IDE-K / KÓDSZERKESZTŐK"},
	{"Geany", "IDE-K / KÓDSZERKESZTŐK"},
	{"Kate", "IDE-K / KÓDSZERKESZTŐK"},
	{"Lite XL", "IDE-K / KÓDSZERKESZTŐK"},
	{"Neovim", "IDE-K / KÓDSZERKESZTŐK"},
	{"Vim", "IDE-K / KÓDSZERKESZTŐK"},
	{"Emacs", "IDE-K / KÓDSZERKESZTŐK"},
	{"Lapce", "IDE-K / KÓDSZERKESZTŐK"},
	{"Helix", "IDE-K / KÓDSZERKESZTŐK"},
	{"Pulsar", "IDE-K / KÓDSZERKESZTŐK"},
	{"Nova", "IDE-K / KÓDSZERKESZTŐK"},
	{"TextMate", "IDE-K / KÓDSZERKESZTŐK"},
	{"JetBrains Toolbox App", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"IntelliJ IDEA Ultimate", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"PyCharm Community", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"PyCharm Professional", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"WebStorm", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"PhpStorm", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"Rider", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"CLion", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"DataGrip", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"GoLand", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"RubyMine", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"RustRover", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"DataSpell", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"Aqua", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"Fleet", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"ReSharper", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"ReSharper C++", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"dotPeek", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"dotCover", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"dotTrace", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"dotMemory", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"TeamCity", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"YouTrack", "JETBRAINS IDE-K / ESZKÖZÖK"},
	{"Fork", "VERZIÓKEZELÉS / GIT"},
	{"Sourcetree", "VERZIÓKEZELÉS / GIT"},
	{"TortoiseGit", "VERZIÓKEZELÉS / GIT"},
	{"TortoiseSVN", "VERZIÓKEZELÉS / GIT"},
	{"Git Extensions", "VERZIÓKEZELÉS / GIT"},
	{"Lazygit", "VERZIÓKEZELÉS / GIT"},
	{"GitHub CLI", "VERZIÓKEZELÉS / GIT"},
	{"GitLab CLI", "VERZIÓKEZELÉS / GIT"},
	{"Gitea", "VERZIÓKEZELÉS / GIT"},
	{"Gitea Actions runner", "VERZIÓKEZELÉS / GIT"},
	{"SmartGit", "VERZIÓKEZELÉS / GIT"},
	{"Tower", "VERZIÓKEZELÉS / GIT"},
	{"Sublime Merge", "VERZIÓKEZELÉS / GIT"},
	{"Plastic SCM / Unity Version Control", "VERZIÓKEZELÉS / GIT"},
	{"Photopea", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Clip Studio Paint", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Corel Painter", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"MediBang Paint", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"FireAlpaca", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Affinity Photo", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Affinity Designer", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Affinity Publisher", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Adobe Photoshop", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Adobe Illustrator", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Adobe Lightroom", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Adobe Fresco", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Adobe Creative Cloud", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Sketchbook", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"ArtRage", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Rebelle", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"PaintTool SAI", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"MyPaint", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Pinta", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Concepts", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"ibis Paint", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Leonardo", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Kleki", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Pixlr", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Fotor", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Canva Desktop", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"PhotoScape X", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"PhotoFiltre", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"PhotoDemon", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"LazPaint", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Krita Gemini / kapcsolódó Krita eszközök", "GRAFIKA / DIGITÁLIS RAJZ"},
	{"Aseprite", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"LibreSprite", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Pixelorama", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Piskel", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"GraphicsGale", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Pyxel Edit", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Pro Motion NG", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Tilesetter", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Tiled Map Editor", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"LDtk", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"TexturePacker", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Shoebox", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"SpriteIlluminator", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Ogmo Editor", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"CrocoTile 3D", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"PixelOver", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Resprite", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"Lospec Pixel Editor", "PIXEL ART / SPRITE / 2D GAME ART"},
	{"FreeCAD", "3D / MODELLEZÉS / CAD"},
	{"OpenSCAD", "3D / MODELLEZÉS / CAD"},
	{"SketchUp", "3D / MODELLEZÉS / CAD"},
	{"Autodesk Fusion", "3D / MODELLEZÉS / CAD"},
	{"Autodesk Maya", "3D / MODELLEZÉS / CAD"},
	{"Autodesk 3ds Max", "3D / MODELLEZÉS / CAD"},
	{"Cinema 4D", "3D / MODELLEZÉS / CAD"},
	{"ZBrush", "3D / MODELLEZÉS / CAD"},
	{"Nomad Sculpt", "3D / MODELLEZÉS / CAD"},
	{"MeshLab", "3D / MODELLEZÉS / CAD"},
	{"Meshmixer", "3D / MODELLEZÉS / CAD"},
	{"Substance 3D Painter", "3D / MODELLEZÉS / CAD"},
	{"Substance 3D Designer", "3D / MODELLEZÉS / CAD"},
	{"Marvelous Designer", "3D / MODELLEZÉS / CAD"},
	{"Houdini", "3D / MODELLEZÉS / CAD"},
	{"Rhino", "3D / MODELLEZÉS / CAD"},
	{"Solid Edge Community Edition", "3D / MODELLEZÉS / CAD"},
	{"SolveSpace", "3D / MODELLEZÉS / CAD"},
	{"BRL-CAD", "3D / MODELLEZÉS / CAD"},
	{"Wings 3D", "3D / MODELLEZÉS / CAD"},
	{"MagicaVoxel", "3D / MODELLEZÉS / CAD"},
	{"Unity Editor", "GAME DEVELOPMENT"},
	{"GameMaker", "GAME DEVELOPMENT"},
	{"Construct 3", "GAME DEVELOPMENT"},
	{"GDevelop", "GAME DEVELOPMENT"},
	{"Defold", "GAME DEVELOPMENT"},
	{"RPG Maker", "GAME DEVELOPMENT"},
	{"Ren'Py", "GAME DEVELOPMENT"},
	{"MonoGame", "GAME DEVELOPMENT"},
	{"Cocos Creator", "GAME DEVELOPMENT"},
	{"CryEngine", "GAME DEVELOPMENT"},
	{"Stride", "GAME DEVELOPMENT"},
	{"Panda3D", "GAME DEVELOPMENT"},
	{"Love2D", "GAME DEVELOPMENT"},
	{"darktable", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"RawTherapee", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"digiKam", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"XnView MP", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"ImageGlass", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"nomacs", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"JPEGView", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"ExifTool", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"Hugin", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"Luminance HDR", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"Photivo", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"Capture One", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"ON1 Photo RAW", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"ACDSee Photo Studio", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"Luminar Neo", "FOTÓ / RAW / KÉPKEZELÉS"},
	{"Flameshot", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Lightshot", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Snipping Tool", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Streamlabs Desktop", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Bandicam", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Camtasia", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"ScreenToGif", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Gyazo", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"LICEcap", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"NVIDIA App / ShadowPlay", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"AMD Software Adrenalin Recording", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Xbox Game Bar", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Action!", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"XSplit Broadcaster", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Elgato 4K Capture Utility", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"Elgato Stream Deck", "SCREENSHOT / KÉPERNYŐRÖGZÍTÉS / STREAM"},
	{"DaVinci Resolve", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Kdenlive", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Shotcut", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"OpenShot", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"FFmpeg", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Avidemux", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"LosslessCut", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Shutter Encoder", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"CapCut Desktop", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"VEGAS Pro", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Adobe Premiere Pro", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Adobe After Effects", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"HitFilm", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Lightworks", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Olive Video Editor", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"VSDC Free Video Editor", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Filmora", "VIDEÓVÁGÁS / KONVERTÁLÁS"},
	{"Ableton Live", "AUDIO / ZENE / DAW"},
	{"Studio One", "AUDIO / ZENE / DAW"},
	{"Cakewalk", "AUDIO / ZENE / DAW"},
	{"Waveform Free", "AUDIO / ZENE / DAW"},
	{"LMMS", "AUDIO / ZENE / DAW"},
	{"Ardour", "AUDIO / ZENE / DAW"},
	{"MuseScore", "AUDIO / ZENE / DAW"},
	{"Equalizer APO", "AUDIO / ZENE / DAW"},
	{"Peace Equalizer", "AUDIO / ZENE / DAW"},
	{"Voicemeeter", "AUDIO / ZENE / DAW"},
	{"VB-CABLE", "AUDIO / ZENE / DAW"},
	{"Mixxx", "AUDIO / ZENE / DAW"},
	{"Ocenaudio", "AUDIO / ZENE / DAW"},
	{"Tenacity", "AUDIO / ZENE / DAW"},
	{"Native Access", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"IK Product Manager", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Arturia Software Center", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Waves Central", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Neural DSP", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"AmpliTube", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Guitar Rig", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"ToneX", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"NAM Universal", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"ReaPlugs", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Helix Native", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"TH-U", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"BIAS FX", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Positive Grid Spark app", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Line 6 Central", "GITÁR / AUDIO PLUGIN / HARDVER"},
	{"Apache OpenOffice", "OFFICE / PDF / JEGYZET"},
	{"Microsoft 365 / Office", "OFFICE / PDF / JEGYZET"},
	{"OnlyOffice Desktop Editors", "OFFICE / PDF / JEGYZET"},
	{"WPS Office", "OFFICE / PDF / JEGYZET"},
	{"FreeOffice", "OFFICE / PDF / JEGYZET"},
	{"PDF24 Creator", "OFFICE / PDF / JEGYZET"},
	{"Foxit PDF Reader", "OFFICE / PDF / JEGYZET"},
	{"Okular", "OFFICE / PDF / JEGYZET"},
	{"Xournal++", "OFFICE / PDF / JEGYZET"},
	{"Obsidian", "OFFICE / PDF / JEGYZET"},
	{"Notion", "OFFICE / PDF / JEGYZET"},
	{"Joplin", "OFFICE / PDF / JEGYZET"},
	{"Logseq", "OFFICE / PDF / JEGYZET"},
	{"Zotero", "OFFICE / PDF / JEGYZET"},
	{"Mendeley Reference Manager", "OFFICE / PDF / JEGYZET"},
	{"Typora", "OFFICE / PDF / JEGYZET"},
	{"MarkText", "OFFICE / PDF / JEGYZET"},
	{"Trilium Notes", "OFFICE / PDF / JEGYZET"},
	{"Standard Notes", "OFFICE / PDF / JEGYZET"},
	{"NanaZip", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Bandizip", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Listary", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Total Commander", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Double Commander", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"FreeCommander", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Files App", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Directory Opus", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"FastCopy", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Bulk Rename Utility", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"SpaceSniffer", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"OneCommander", "FÁJLKEZELÉS / KERESÉS / TÖMÖRÍTÉS"},
	{"Sysinternals Suite", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Process Explorer", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Process Monitor", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Autoruns", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"TCPView", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"GPU-Z", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"HWMonitor", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Speccy", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Hard Disk Sentinel", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"RivaTuner Statistics Server", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"OCCT", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Prime95", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Cinebench", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"MemTest86", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Victoria HDD/SSD", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"AIDA64", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Core Temp", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Open Hardware Monitor", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"Libre Hardware Monitor", "RENDSZER / HARDVER / DIAGNOSZTIKA"},
	{"UNetbootin", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"YUMI", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"Win32 Disk Imager", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"PowerISO", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"AnyBurn", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"WinCDEmu", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"Virtual CloneDrive", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"Media Creation Tool", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"Windows Installation Assistant", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"NTLite", "WINDOWS TELEPÍTÉS / USB / ISO"},
	{"Hyper-V", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"WSL2 disztribúciók", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"QEMU", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"Vagrant", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"Multipass", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"Minikube", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"Kubernetes kubectl", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"kind", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"k3d", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"Portainer Desktop / kapcsolódó kliensek", "VIRTUALIZÁCIÓ / KONTAINEREK"},
	{"GNS3", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"EVE-NG Client Pack", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"KiTTY", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"Angry IP Scanner", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"Advanced IP Scanner", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"PortQry", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"iperf3", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"OpenSpeedTest", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"Netcat / Ncat", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"Solar-PuTTY", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"Bitvise SSH Client", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"OpenSSH for Windows", "HÁLÓZATI / CISCO / IT ESZKÖZÖK"},
	{"MEGA Desktop App", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Proton Drive", "FELHŐ / SZINKRONIZÁLÁS"},
	{"pCloud Drive", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Nextcloud Desktop", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Syncthing", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Resilio Sync", "FELHŐ / SZINKRONIZÁLÁS"},
	{"rclone", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Cyberduck", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Mountain Duck", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Box Drive", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Koofr", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Seafile Client", "FELHŐ / SZINKRONIZÁLÁS"},
	{"Bitwarden", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"1Password", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"KeePass", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"KeePassXC", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Proton Pass", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Enpass", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Authy", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"2FAS", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Aegis (mobil)", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Yubico Authenticator", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Windows Defender", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"ESET", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Sophos Home", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"GlassWire", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"SimpleWall", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Portmaster", "JELSZÓ / 2FA / BIZTONSÁG"},
	{"Slack", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Signal Desktop", "EMAIL / KOMMUNIKÁCIÓ"},
	{"WhatsApp Desktop", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Viber", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Element", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Mattermost Desktop", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Rocket.Chat Desktop", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Thunderbird", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Mailspring", "EMAIL / KOMMUNIKÁCIÓ"},
	{"eM Client", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Spark Desktop", "EMAIL / KOMMUNIKÁCIÓ"},
	{"Proton Mail Bridge", "EMAIL / KOMMUNIKÁCIÓ"},
	{"LibreWolf", "BÖNGÉSZŐK"},
	{"Floorp", "BÖNGÉSZŐK"},
	{"Waterfox", "BÖNGÉSZŐK"},
	{"Tor Browser", "BÖNGÉSZŐK"},
	{"Arc", "BÖNGÉSZŐK"},
	{"Zen Browser", "BÖNGÉSZŐK"},
	{"Chromium", "BÖNGÉSZŐK"},
	{"Free Download Manager", "DOWNLOAD MANAGEREK"},
	{"JDownloader 2", "DOWNLOAD MANAGEREK"},
	{"Internet Download Manager", "DOWNLOAD MANAGEREK"},
	{"Xtreme Download Manager", "DOWNLOAD MANAGEREK"},
	{"Motrix", "DOWNLOAD MANAGEREK"},
	{"aria2", "DOWNLOAD MANAGEREK"},
	{"Persepolis Download Manager", "DOWNLOAD MANAGEREK"},
	{"yt-dlp", "DOWNLOAD MANAGEREK"},
	{"Gallery-dl", "DOWNLOAD MANAGEREK"},
	{"Transmission", "TORRENT"},
	{"Deluge", "TORRENT"},
	{"BiglyBT", "TORRENT"},
	{"Tixati", "TORRENT"},
	{"Tribler", "TORRENT"},
}
