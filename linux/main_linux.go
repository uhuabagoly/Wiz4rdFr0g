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

	"wiz4rdfr0g.local/fullcatalog/internal/linuxcatalog"
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

type linuxCandidate = linuxcatalog.Candidate

type linuxApp struct {
	linuxCandidate
	Provider      string
	Package       string
	FlatpakRemote string
	InstallPath   string
	VendorDEB     bool
	PackageFile   string
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

var candidates = linuxcatalog.Candidates

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
	if len(os.Args) == 4 && os.Args[1] == "--physical-test" {
		os.Exit(linuxLifecycleArgs())
	}
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
	b, err := exec.Command("flatpak", "remote-ls", "--user", "--app", "--columns=origin,application").Output()
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
		if c.Name == "Python 3" {
			// Manage an independent interpreter; the distribution's system
			// Python is an OS dependency and is not an uninstallable app.
			if _, err := exec.LookPath("uv"); err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				rows, err := managedPythonDownloads(ctx)
				cancel()
				if selected, ok := newestManagedPython(rows); err == nil && ok {
					add(linuxApp{linuxCandidate: c, Provider: "uv-python", Package: selected.Key, InstallPath: "uv által kezelt, külön Python-telepítés"})
				}
			}
			continue
		}
		// These suite metapackages leave the application payload installed
		// after apt remove. Prefer their complete, self-contained provider.
		if c.Name == "VLC Media Player" || c.Name == "LibreOffice" {
			found := false
			for _, id := range c.Flatpak {
				if remote, ok := flatpakSet[id]; ok {
					add(linuxApp{linuxCandidate: c, Provider: "flatpak", Package: id, FlatpakRemote: remote, InstallPath: "Felhasználói Flatpak (~/.local/share/flatpak)"})
					found = true
					break
				}
			}
			if found {
				continue
			}
		}
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
			transitional := false
			if packageManager == "apt-get" {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if p == "postgresql" || p == "emacs" || p == "qemu-system" {
					dependencies := exec.CommandContext(ctx, "apt-cache", "depends", p)
					dependencies.Env = append(os.Environ(), "LC_ALL=C")
					if output, err := dependencies.Output(); err == nil {
						p = linuxpkg.APTPayloadPackage(p, string(output), packageSet)
					}
				}
				cmd := exec.CommandContext(ctx, "apt-cache", "policy", p)
				cmd.Env = append(os.Environ(), "LC_ALL=C")
				policy, err := cmd.Output()
				cancel()
				v, candidateErr := linuxpkg.APTCandidate(string(policy))
				transitional = err != nil || candidateErr != nil || strings.Contains(v, "snap")
			}
			if !transitional {
				add(linuxApp{linuxCandidate: c, Provider: packageManager, Package: p, InstallPath: "Rendszer által kezelt (/usr, /opt, ... )"})
				continue
			}
		}
		if c.Name == "VeraCrypt" && packageManager == "apt-get" {
			if _, err := veraCryptPlatform(); err == nil {
				add(linuxApp{linuxCandidate: c, Provider: "apt-get", Package: "veracrypt", VendorDEB: true, InstallPath: "Hivatalos, PGP-ellenőrzött VeraCrypt DEB"})
				continue
			}
		}
		for _, p := range c.Flatpak {
			if remote, ok := flatpakSet[p]; ok {
				add(linuxApp{linuxCandidate: c, Provider: "flatpak", Package: p, FlatpakRemote: remote, InstallPath: "Felhasználói Flatpak (~/.local/share/flatpak)"})
				break
			}
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
	if a.VendorDEB {
		if a.Name != "VeraCrypt" || a.Package != "veracrypt" || a.Provider != "apt-get" {
			return linuxpkg.Spec{}, fmt.Errorf("unsupported vendor DEB identity")
		}
		path := a.PackageFile
		if path == "" {
			directory, err := os.MkdirTemp("", "Wiz4rdFr0g-veracrypt-")
			if err != nil {
				return linuxpkg.Spec{}, err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			path, _, err = prepareVeraCryptDeb(ctx, directory)
			if err != nil {
				os.RemoveAll(directory)
				return linuxpkg.Spec{}, err
			}
		}
		return linuxpkg.Spec{Name: "apt-get", Args: []string{"install", "-y", path}, NeedsRoot: true}, nil
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
	if err := linuxpkg.Validate(a.Provider, a.Package, a.FlatpakRemote, ""); err != nil {
		return false, err
	}
	spec, err := linuxpkg.Inventory(a.Provider)
	if err != nil {
		return false, err
	}
	out, err := exec.CommandContext(ctx, spec.Name, spec.Args...).Output()
	if err != nil {
		return false, fmt.Errorf("installed state unknown: %w", err)
	}
	return linuxpkg.InventoryContains(a.Provider, a.Package, out, 0)
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
