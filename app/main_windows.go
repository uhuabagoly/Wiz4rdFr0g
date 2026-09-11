//go:build windows

package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

type appState struct {
	Selected             bool
	Version              string
	InstallDir           string
	Versions             []string
	ResolvedID           string
	ResolvedSource       string
	ResolveError         string
	VersionsLoaded       bool
	VersionsLoading      bool
	VersionsStale        bool
	Installed            bool
	InstalledVersion     string
	InstalledID          string
	InstalledName        string
	InstalledRegistryKey string
	InstalledDisplayIcon string
	ActualInstallDir     string
	UninstallString      string
	QuietUninstallString string
	InstallScope         string
	WindowsInstaller     bool
	CanUninstall         bool
	InstalledMatchType   string
	InstalledConfidence  int
	UninstallStrategy    string
}

type rowUI struct {
	catalogIndex                                                  int
	checkbox, nameLabel, combo, installEdit, browseBtn, deleteBtn uintptr
}

type selection struct {
	CatalogIndex                          int
	ID, Source, Name, Version, InstallDir string
	CustomInstallDir                      bool
}

type resultInfo struct {
	DownloadOK, DownloadSkipped, InstallOK bool
	Error                                  string
}

type installerMetadata struct {
	SourcePath string `json:"sourcePath"`
	SHA256     string `json:"sha256"`
	Publisher  string `json:"publisher"`
	Version    string `json:"version"`
}

type installedPackage struct {
	Name                 string
	ID                   string
	Version              string
	Source               string
	InstallLocation      string
	DisplayIcon          string
	UninstallString      string
	QuietUninstallString string
	RegistryKey          string
	Scope                string
	WindowsInstaller     bool
	CanUninstall         bool
}

type registryPackage struct {
	DisplayName          string `json:"DisplayName"`
	DisplayVersion       string `json:"DisplayVersion"`
	InstallLocation      string `json:"InstallLocation"`
	DisplayIcon          string `json:"DisplayIcon"`
	UninstallString      string `json:"UninstallString"`
	QuietUninstallString string `json:"QuietUninstallString"`
	WindowsInstaller     int    `json:"WindowsInstaller"`
	RegistryKey          string `json:"RegistryKey"`
	Scope                string `json:"Scope"`
}

type packageCacheEntry struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	UpdatedAt       int64  `json:"updatedAt"`
	ResolverVersion int    `json:"resolverVersion"`
}

type versionCacheEntry struct {
	ID        string   `json:"id"`
	Source    string   `json:"source"`
	Versions  []string `json:"versions"`
	UpdatedAt int64    `json:"updatedAt"`
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW    = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procLoadImageW           = user32.NewProc("LoadImageW")
	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procWaitForSingleObject  = kernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess   = kernel32.NewProc("GetExitCodeProcess")
	procTerminateProcess     = kernel32.NewProc("TerminateProcess")
	procCloseHandle          = kernel32.NewProc("CloseHandle")
	procGetStockObject       = gdi32.NewProc("GetStockObject")
	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procShellExecuteExW      = shell32.NewProc("ShellExecuteExW")
	procShellExecuteW        = shell32.NewProc("ShellExecuteW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
)

const (
	appTitle               = "Wiz4rd Fr0g"
	appVersion             = releaseproof.AppVersion
	publisher              = "https://github.com/uhuabagoly"
	packageResolverVersion = 2

	visibleRows = 16

	WS_CAPTION              = 0x00C00000
	WS_SYSMENU              = 0x00080000
	WS_MINIMIZEBOX          = 0x00020000
	WS_FIXEDWINDOW          = WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX
	WS_VISIBLE              = 0x10000000
	WS_CHILD                = 0x40000000
	WS_TABSTOP              = 0x00010000
	WS_BORDER               = 0x00800000
	WS_VSCROLL              = 0x00200000
	WS_HSCROLL              = 0x00100000
	BS_PUSHBUTTON           = 0
	BS_AUTOCHECKBOX         = 3
	CBS_DROPDOWNLIST        = 3
	CBS_HASSTRINGS          = 0x200
	CBS_NOINTEGRALHEIGHT    = 0x400
	ES_MULTILINE            = 0x4
	ES_AUTOVSCROLL          = 0x40
	ES_AUTOHSCROLL          = 0x80
	ES_READONLY             = 0x800
	WM_DESTROY              = 0x0002
	WM_COMMAND              = 0x0111
	WM_SYSCOMMAND           = 0x0112
	WM_SETFONT              = 0x0030
	WM_APP_UI               = 0x8001
	BM_GETCHECK             = 0x00F0
	BM_SETCHECK             = 0x00F1
	BST_UNCHECKED           = 0
	BST_CHECKED             = 1
	CB_ADDSTRING            = 0x0143
	CB_RESETCONTENT         = 0x014B
	CB_GETCURSEL            = 0x0147
	CB_SETCURSEL            = 0x014E
	CB_GETLBTEXT            = 0x0148
	CB_GETLBTEXTLEN         = 0x0149
	CBN_SELCHANGE           = 1
	EM_SETSEL               = 0x00B1
	EM_REPLACESEL           = 0x00C2
	BN_CLICKED              = 0
	EN_CHANGE               = 0x0300
	SC_SIZE                 = 0xF000
	SC_MAXIMIZE             = 0xF030
	MB_OK                   = 0
	MB_YESNO                = 4
	MB_ICONERROR            = 0x10
	MB_ICONQUESTION         = 0x20
	MB_ICONWARNING          = 0x30
	MB_ICONINFO             = 0x40
	IDYES                   = 6
	SW_HIDE                 = 0
	SW_SHOW                 = 5
	SWP_NOZORDER            = 0x0004
	SWP_NOACTIVATE          = 0x0010
	IMAGE_ICON              = 1
	LR_DEFAULTCOLOR         = 0x0000
	BIF_RETURNONLYFSDIRS    = 0x0001
	BIF_NEWDIALOGSTYLE      = 0x0040
	SEE_MASK_NOCLOSEPROCESS = 0x00000040
	WAIT_OBJECT_0           = 0x00000000
	WAIT_TIMEOUT            = 0x00000102
	SW_SHOWNORMAL           = 1

	ID_SEARCH            = 1001
	ID_CLEAR_SEARCH      = 1002
	ID_DOWNLOAD_DIR      = 1003
	ID_BROWSE_DOWNLOAD   = 1004
	ID_NEXT1             = 1005
	ID_PREV_CATALOG      = 1006
	ID_NEXT_CATALOG      = 1007
	ID_BACK2             = 1010
	ID_INSTALL           = 1011
	ID_CANCEL            = 1012
	ID_NEXT2             = 1013
	ID_DELETE_SETUP      = 1020
	ID_CLOSE             = 1021
	ID_CHECK_BASE        = 1100
	ID_COMBO_BASE        = 1200
	ID_INSTALL_EDIT_BASE = 1400
	ID_BROWSE_BASE       = 1500
	ID_DELETE_BASE       = 1600
)

type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIconOrMonitor uintptr
	hProcess       uintptr
}

type wndclassex struct {
	cbSize, style                            uint32
	lpfnWndProc                              uintptr
	cbClsExtra, cbWndExtra                   int32
	hInstance, hIcon, hCursor, hbrBackground uintptr
	lpszMenuName, lpszClassName              *uint16
	hIconSm                                  uintptr
}
type point struct{ x, y int32 }
type msg struct {
	hwnd           uintptr
	message        uint32
	wParam, lParam uintptr
	time           uint32
	pt             point
	private        uint32
}
type browseInfo struct {
	hwndOwner, pidlRoot       uintptr
	pszDisplayName, lpszTitle *uint16
	ulFlags                   uint32
	lpfn                      uintptr
	lParam                    uintptr
	iImage                    int32
}

var (
	mainWnd, defaultFont uintptr

	searchEdit, downloadDirEdit, resultLabel, prevCatalogBtn, nextCatalogBtn uintptr
	page1Controls, page2Controls, page3Controls                              []uintptr
	rows                                                                     []*rowUI

	summaryEdit, installBtn, cancelBtn, next2Btn, back2Btn, logEdit, operationLabel uintptr
	finishSummaryEdit, cleanupInfo, deleteSetupBtn                                  uintptr

	states          []appState
	filteredIndices []int
	catalogPage     int
	selected        []selection
	results         map[int]*resultInfo
	statusByIndex   map[int]string
	currentPage     = 1
	wingetOK        bool

	stateMu            sync.Mutex
	busy               bool
	currentCancel      context.CancelFunc
	uiMu               sync.Mutex
	uiQueue            []func()
	versionRegexp      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+\-]*$`)
	splitColsRe        = regexp.MustCompile(`\s{2,}`)
	versionSem         = make(chan struct{}, 6)
	packageCache       = map[string]packageCacheEntry{}
	cacheMu            sync.Mutex
	versionCache       = map[string]versionCacheEntry{}
	versionCacheMu     sync.Mutex
	versionCacheSaveMu sync.Mutex
	searchDebounceMu   sync.Mutex
	searchDebounceSeq  uint64
	sourceRefreshOnce  sync.Once
)

func main() {
	if vmTestRequested() {
		os.Exit(runVMTestFromArgs())
	}
	if len(os.Args) >= 3 {
		idx, err := strconv.Atoi(os.Args[2])
		if err == nil {
			switch os.Args[1] {
			case "--uninstall-worker-user":
				os.Exit(runUninstallWorker(idx, false))
			case "--uninstall-worker-admin", "--uninstall-worker":
				os.Exit(runUninstallWorker(idx, true))
			case "--close-running-admin":
				os.Exit(runCloseRunningWorker(idx))
			}
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, 2)
	defer procCoUninitialize.Call()
	runGUI()
}

func runGUI() {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className := utf16Ptr("Wiz4rdFr0gNativeWizard")
	hIcon, _, _ := procLoadImageW.Call(hInstance, 1, IMAGE_ICON, 32, 32, LR_DEFAULTCOLOR)
	hIconSm, _, _ := procLoadImageW.Call(hInstance, 1, IMAGE_ICON, 16, 16, LR_DEFAULTCOLOR)
	wc := wndclassex{cbSize: uint32(unsafe.Sizeof(wndclassex{})), lpfnWndProc: syscall.NewCallback(windowProc), hInstance: hInstance, hIcon: hIcon, hIconSm: hIconSm, hbrBackground: 6, lpszClassName: className}
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		messageBox(0, "Az ablakosztály nem hozható létre.", appTitle, MB_OK|MB_ICONERROR)
		return
	}
	mainWnd = createWindow(0, "Wiz4rdFr0gNativeWizard", appTitle, WS_FIXEDWINDOW, 70, 35, 1225, 910, 0, 0, hInstance)
	if mainWnd == 0 {
		messageBox(0, "Az alkalmazásablak nem hozható létre.", appTitle, MB_OK|MB_ICONERROR)
		return
	}
	defaultFont, _, _ = procGetStockObject.Call(17)
	initCatalogState()
	buildUI(hInstance)
	refreshFilter(true)
	setPage(1)
	procShowWindow.Call(mainWnd, SW_SHOW)
	procUpdateWindow.Call(mainWnd)
	go checkWinget()
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func initCatalogState() {
	loadPackageCache()
	loadVersionCache()
	states = make([]appState, len(catalog))
	for i, a := range catalog {
		states[i].InstallDir = defaultInstallDir(a)
		states[i].ResolvedID = a.ID
		if a.ID != "" {
			states[i].ResolvedSource = "winget"
			applyCachedVersionsToState(i)
			continue
		}
		if cached, ok := getCachedPackage(a.Name); ok {
			states[i].ResolvedID = cached.ID
			states[i].ResolvedSource = cached.Source
			applyCachedVersionsToState(i)
		}
	}
}

func applyCachedVersionsToState(idx int) {
	if idx < 0 || idx >= len(states) {
		return
	}
	st := &states[idx]
	if st.ResolvedID == "" || st.ResolvedSource == "" || st.ResolvedSource == "msstore" {
		return
	}
	entry, ok := getCachedVersions(st.ResolvedID, st.ResolvedSource)
	if !ok {
		return
	}
	st.Versions = append([]string(nil), entry.Versions...)
	st.VersionsLoaded = true
	age := time.Since(time.Unix(entry.UpdatedAt, 0))
	st.VersionsStale = age > versionCacheFreshTTL
}

func buildUI(h uintptr) {
	addStatic("Keresés:", 20, 22, 70, 24, h, &page1Controls)
	searchEdit = createWindow(0, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP, 90, 20, 700, 26, mainWnd, ID_SEARCH, h)
	setFont(searchEdit)
	page1Controls = append(page1Controls, searchEdit)
	addButton("Keresés törlése", 800, 19, 130, 28, ID_CLEAR_SEARCH, h, &page1Controls)
	addStatic("Telepítőcsomagok letöltési helye:", 20, 58, 250, 24, h, &page1Controls)
	downloadDirEdit = createWindow(0, "EDIT", defaultDownloadDir(), WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP, 270, 56, 650, 26, mainWnd, ID_DOWNLOAD_DIR, h)
	setFont(downloadDirEdit)
	page1Controls = append(page1Controls, downloadDirEdit)
	addButton("Tallózás...", 930, 55, 100, 28, ID_BROWSE_DOWNLOAD, h, &page1Controls)
	addStatic("Program", 48, 92, 235, 22, h, &page1Controls)
	addStatic("Verzió", 295, 92, 195, 22, h, &page1Controls)
	addStatic("Telepítési hely", 505, 92, 390, 22, h, &page1Controls)

	rows = make([]*rowUI, visibleRows)
	for i := 0; i < visibleRows; i++ {
		y := 116 + i*34
		check := createWindow(0, "BUTTON", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 20, y+2, 22, 22, mainWnd, uintptr(ID_CHECK_BASE+i), h)
		setFont(check)
		page1Controls = append(page1Controls, check)
		name := addStatic("", 48, y+3, 235, 22, h, &page1Controls)
		combo := createWindow(0, "COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST|CBS_HASSTRINGS|CBS_NOINTEGRALHEIGHT, 295, y, 195, 230, mainWnd, uintptr(ID_COMBO_BASE+i), h)
		setFont(combo)
		page1Controls = append(page1Controls, combo)
		edit := createWindow(0, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP, 505, y, 495, 26, mainWnd, uintptr(ID_INSTALL_EDIT_BASE+i), h)
		setFont(edit)
		page1Controls = append(page1Controls, edit)
		browse := addButton("Tallózás", 1008, y, 86, 27, uintptr(ID_BROWSE_BASE+i), h, &page1Controls)
		deleteBtn := addButton("Törlés", 1100, y, 75, 27, uintptr(ID_DELETE_BASE+i), h, &page1Controls)
		procShowWindow.Call(deleteBtn, SW_HIDE)
		rows[i] = &rowUI{catalogIndex: -1, checkbox: check, nameLabel: name, combo: combo, installEdit: edit, browseBtn: browse, deleteBtn: deleteBtn}
	}
	resultLabel = addStatic("", 20, 692, 440, 24, h, &page1Controls)
	prevCatalogBtn = addButton("Előző oldal", 470, 686, 110, 32, ID_PREV_CATALOG, h, &page1Controls)
	nextCatalogBtn = addButton("Következő oldal", 590, 686, 125, 32, ID_NEXT_CATALOG, h, &page1Controls)
	addButton("Tovább", 1055, 686, 120, 34, ID_NEXT1, h, &page1Controls)

	addStatic("Telepítés", 20, 22, 200, 26, h, &page2Controls)
	addStatic("A Telepít gomb után a kijelölt csomagok letöltése és telepítése egymás után történik.", 20, 50, 920, 22, h, &page2Controls)
	addStatic("Kijelölt programok / állapot", 20, 87, 320, 22, h, &page2Controls)
	summaryEdit = createWindow(0, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_READONLY, 20, 112, 760, 579, mainWnd, 0, h)
	setFont(summaryEdit)
	page2Controls = append(page2Controls, summaryEdit)
	operationLabel = addStatic("Telepítésre vár.", 810, 87, 365, 22, h, &page2Controls)
	installBtn = addButton("Telepít", 810, 117, 120, 34, ID_INSTALL, h, &page2Controls)
	cancelBtn = addButton("Megszakítás", 940, 117, 120, 34, ID_CANCEL, h, &page2Controls)
	procEnableWindow.Call(cancelBtn, 0)
	addStatic("Műveleti napló / konzol", 810, 162, 300, 22, h, &page2Controls)
	logEdit = createWindow(0, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_READONLY, 810, 188, 365, 503, mainWnd, 0, h)
	setFont(logEdit)
	page2Controls = append(page2Controls, logEdit)
	back2Btn = addButton("Vissza", 20, 700, 120, 34, ID_BACK2, h, &page2Controls)
	next2Btn = addButton("Tovább", 1055, 700, 120, 34, ID_NEXT2, h, &page2Controls)
	procEnableWindow.Call(next2Btn, 0)

	addStatic("Befejezés", 20, 22, 200, 28, h, &page3Controls)
	addStatic("Eredmény:", 20, 60, 120, 22, h, &page3Controls)
	finishSummaryEdit = createWindow(0, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 20, 86, 1155, 450, mainWnd, 0, h)
	setFont(finishSummaryEdit)
	page3Controls = append(page3Controls, finishSummaryEdit)
	addStatic("Wiz4rd Fr0g saját telepítőcsomag:", 20, 560, 300, 22, h, &page3Controls)
	cleanupInfo = addStatic("Ellenőrzés...", 20, 588, 900, 42, h, &page3Controls)
	deleteSetupBtn = addButton("Setup.exe törlése", 940, 582, 180, 34, ID_DELETE_SETUP, h, &page3Controls)
	procEnableWindow.Call(deleteSetupBtn, 0)
	addStatic("A törlés csak a Wiz4rd Fr0g eredeti Setup.exe fájljára vonatkozik; a telepített programokat nem érinti.", 20, 640, 980, 24, h, &page3Controls)
	addButton("Bezárás", 1055, 690, 120, 34, ID_CLOSE, h, &page3Controls)
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case WM_SYSCOMMAND:
		sc := wParam & 0xFFF0
		if sc == SC_SIZE || sc == SC_MAXIMIZE {
			return 0
		}
	case WM_APP_UI:
		drainUITasks()
		return 0
	case WM_COMMAND:
		id := int(wParam & 0xffff)
		code := int((wParam >> 16) & 0xffff)

		if id == ID_SEARCH && code == EN_CHANGE {
			scheduleFilterRefresh()
			return 0
		}
		if id >= ID_INSTALL_EDIT_BASE && id < ID_INSTALL_EDIT_BASE+visibleRows && code == EN_CHANGE {
			slot := id - ID_INSTALL_EDIT_BASE
			if validSlot(slot) {
				idx := rows[slot].catalogIndex
				states[idx].InstallDir = strings.TrimSpace(getText(rows[slot].installEdit))
			}
			return 0
		}
		if id >= ID_COMBO_BASE && id < ID_COMBO_BASE+visibleRows && code == CBN_SELCHANGE {
			slot := id - ID_COMBO_BASE
			if validSlot(slot) {
				idx := rows[slot].catalogIndex
				v := comboSelectedText(rows[slot].combo)
				if v == "Legújabb" {
					v = ""
				}
				if v != "Verziók betöltése..." && v != "Nem elérhető Wingetből" && v != "Winget ellenőrzése..." {
					states[idx].Version = v
				}
			}
			return 0
		}
		if code == BN_CLICKED {
			if id >= ID_DELETE_BASE && id < ID_DELETE_BASE+visibleRows {
				slot := id - ID_DELETE_BASE
				if validSlot(slot) {
					uninstallCatalogProgram(rows[slot].catalogIndex)
				}
				return 0
			}
			if id >= ID_CHECK_BASE && id < ID_CHECK_BASE+visibleRows {
				slot := id - ID_CHECK_BASE
				if validSlot(slot) {
					idx := rows[slot].catalogIndex
					states[idx].Selected = checkboxChecked(rows[slot].checkbox)
					updateCatalogFooter()
				}
				return 0
			}
			switch {
			case id == ID_CLEAR_SEARCH:
				setText(searchEdit, "")
				refreshFilter(true)
				return 0
			case id == ID_BROWSE_DOWNLOAD:
				if p := browseFolder(getText(downloadDirEdit)); p != "" {
					setText(downloadDirEdit, p)
				}
				return 0
			case id == ID_PREV_CATALOG:
				if catalogPage > 0 {
					catalogPage--
					bindCatalogPage()
				}
				return 0
			case id == ID_NEXT_CATALOG:
				if (catalogPage+1)*visibleRows < len(filteredIndices) {
					catalogPage++
					bindCatalogPage()
				}
				return 0
			case id == ID_NEXT1:
				goStep2()
				return 0
			case id == ID_BACK2:
				if isBusy() {
					if messageBox(mainWnd, "A telepítés még folyamatban van. Megszakítod és visszalépsz a kiválasztáshoz?", appTitle, MB_YESNO|MB_ICONQUESTION) != IDYES {
						return 0
					}
					cancelOperation()
				}
				setPage(1)
				return 0
			case id == ID_INSTALL:
				startInstall()
				return 0
			case id == ID_CANCEL:
				cancelOperation()
				return 0
			case id == ID_NEXT2:
				if !isBusy() {
					buildFinish()
					setPage(3)
				}
				return 0
			case id == ID_DELETE_SETUP:
				deleteSetup()
				return 0
			case id == ID_CLOSE:
				procPostMessageW.Call(mainWnd, WM_DESTROY, 0, 0)
				return 0
			case id >= ID_BROWSE_BASE && id < ID_BROWSE_BASE+visibleRows:
				slot := id - ID_BROWSE_BASE
				if validSlot(slot) {
					idx := rows[slot].catalogIndex
					if p := browseFolder(getText(rows[slot].installEdit)); p != "" {
						states[idx].InstallDir = p
						setText(rows[slot].installEdit, p)
					}
				}
				return 0
			}
		}
	case WM_DESTROY:
		cancelOperation()
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func validSlot(slot int) bool {
	return slot >= 0 && slot < len(rows) && rows[slot] != nil && rows[slot].catalogIndex >= 0 && rows[slot].catalogIndex < len(catalog)
}

func setPage(page int) {
	currentPage = page
	showGroup(page1Controls, page == 1)
	showGroup(page2Controls, page == 2)
	showGroup(page3Controls, page == 3)
	if page == 1 {
		bindCatalogPage()
	}
}

func showGroup(group []uintptr, show bool) {
	cmd := uintptr(SW_HIDE)
	if show {
		cmd = SW_SHOW
	}
	for _, h := range group {
		if h != 0 {
			procShowWindow.Call(h, cmd)
		}
	}
}

func scheduleFilterRefresh() {
	searchDebounceMu.Lock()
	searchDebounceSeq++
	seq := searchDebounceSeq
	searchDebounceMu.Unlock()
	go func() {
		time.Sleep(260 * time.Millisecond)
		searchDebounceMu.Lock()
		current := searchDebounceSeq
		searchDebounceMu.Unlock()
		if seq != current {
			return
		}
		uiDo(func() {
			searchDebounceMu.Lock()
			stillCurrent := seq == searchDebounceSeq
			searchDebounceMu.Unlock()
			if stillCurrent {
				refreshFilter(true)
			}
		})
	}()
}

func refreshFilter(resetPage bool) {
	q := normalizeSearch(getText(searchEdit))
	filteredIndices = filteredIndices[:0]
	installed := make([]int, 0)
	available := make([]int, 0)
	for i, a := range catalog {
		hay := normalizeSearch(a.Name + " " + a.Category + " " + a.ID)
		if q != "" && !strings.Contains(hay, q) {
			continue
		}
		if states[i].Installed {
			installed = append(installed, i)
		} else {
			available = append(available, i)
		}
	}
	filteredIndices = append(filteredIndices, installed...)
	filteredIndices = append(filteredIndices, available...)
	if resetPage {
		catalogPage = 0
	}
	maxPage := 0
	if len(filteredIndices) > 0 {
		maxPage = (len(filteredIndices) - 1) / visibleRows
	}
	if catalogPage > maxPage {
		catalogPage = maxPage
	}
	bindCatalogPage()
}

func bindCatalogPage() {
	if currentPage != 1 {
		for _, r := range rows {
			if r == nil {
				continue
			}
			for _, h := range []uintptr{r.checkbox, r.nameLabel, r.combo, r.installEdit, r.browseBtn, r.deleteBtn} {
				procShowWindow.Call(h, SW_HIDE)
			}
		}
		return
	}
	start := catalogPage * visibleRows
	for slot := 0; slot < visibleRows; slot++ {
		pos := start + slot
		r := rows[slot]
		if pos >= len(filteredIndices) {
			r.catalogIndex = -1
			for _, h := range []uintptr{r.checkbox, r.nameLabel, r.combo, r.installEdit, r.browseBtn, r.deleteBtn} {
				procShowWindow.Call(h, SW_HIDE)
			}
			continue
		}
		idx := filteredIndices[pos]
		r.catalogIndex = idx
		a := catalog[idx]
		st := &states[idx]
		for _, h := range []uintptr{r.checkbox, r.nameLabel, r.combo, r.installEdit, r.browseBtn} {
			procShowWindow.Call(h, SW_SHOW)
		}
		if st.Installed && st.CanUninstall {
			procShowWindow.Call(r.deleteBtn, SW_SHOW)
			procEnableWindow.Call(r.deleteBtn, 1)
		} else {
			procShowWindow.Call(r.deleteBtn, SW_HIDE)
		}
		setText(r.nameLabel, shorten(a.Name, 35))
		procSendMessageW.Call(r.checkbox, BM_SETCHECK, boolCheck(st.Selected), 0)
		setText(r.installEdit, st.InstallDir)
		populateRowCombo(slot)
		if wingetOK && (!st.VersionsLoaded || st.VersionsStale) && !st.VersionsLoading {
			ensureVersionsLoaded(idx)
		}
	}
	updateCatalogFooter()
}

func populateRowCombo(slot int) {
	if !validSlot(slot) {
		return
	}
	idx := rows[slot].catalogIndex
	r := rows[slot]
	st := &states[idx]
	comboReset(r.combo)
	switch {
	case !wingetOK:
		comboAdd(r.combo, "Winget ellenőrzése...")
		comboSelect(r.combo, 0)
		procEnableWindow.Call(r.combo, 0)
	case st.VersionsLoaded || len(st.Versions) > 0:

		procEnableWindow.Call(r.checkbox, 1)
		comboAdd(r.combo, "Legújabb")
		selectedPos := 0
		for i, v := range st.Versions {
			comboAdd(r.combo, v)
			if st.Version == v {
				selectedPos = i + 1
			}
		}
		comboSelect(r.combo, selectedPos)
		procEnableWindow.Call(r.combo, 1)
	case st.VersionsLoading:
		comboAdd(r.combo, "Legújabb")
		comboAdd(r.combo, "Csomag ellenőrzése...")
		comboSelect(r.combo, 0)
		procEnableWindow.Call(r.combo, 0)
		procEnableWindow.Call(r.checkbox, 0)
	case st.ResolveError != "":
		comboAdd(r.combo, "Nem elérhető")
		comboSelect(r.combo, 0)
		procEnableWindow.Call(r.combo, 0)
		st.Selected = false
		procSendMessageW.Call(r.checkbox, BM_SETCHECK, BST_UNCHECKED, 0)
		procEnableWindow.Call(r.checkbox, 0)
	default:
		comboAdd(r.combo, "Ellenőrzés...")
		comboSelect(r.combo, 0)
		procEnableWindow.Call(r.combo, 0)
		procEnableWindow.Call(r.checkbox, 0)
	}
}

func updateCatalogFooter() {
	selectedCount := 0
	for i := range states {
		if states[i].Selected {
			selectedCount++
		}
	}
	pages := 1
	if len(filteredIndices) > 0 {
		pages = (len(filteredIndices) + visibleRows - 1) / visibleRows
	}
	page := catalogPage + 1
	if page > pages {
		page = pages
	}
	setText(resultLabel, fmt.Sprintf("%d találat | %d kijelölve | %d/%d oldal | teljes katalógus: %d", len(filteredIndices), selectedCount, page, pages, len(catalog)))
	procEnableWindow.Call(prevCatalogBtn, boolPtr(catalogPage > 0))
	procEnableWindow.Call(nextCatalogBtn, boolPtr((catalogPage+1)*visibleRows < len(filteredIndices)))
}

func ensureVersionsLoaded(idx int) {
	if idx < 0 || idx >= len(catalog) || states[idx].VersionsLoading || !wingetOK {
		return
	}
	if states[idx].VersionsLoaded && !states[idx].VersionsStale {
		return
	}
	states[idx].VersionsLoading = true
	updateVisibleCatalogItem(idx)
	app := catalog[idx]
	go func() {
		versionSem <- struct{}{}
		id, source, versions, err := loadPackageData(app)
		<-versionSem
		uiDo(func() {
			st := &states[idx]
			st.VersionsLoading = false
			if err != nil {

				if len(st.Versions) > 0 {
					st.VersionsLoaded = true
					st.VersionsStale = true
					st.ResolveError = ""
				} else {
					st.VersionsLoaded = true
					st.VersionsStale = false
					st.ResolveError = err.Error()
				}
			} else {
				st.VersionsLoaded = true
				st.VersionsStale = false
				st.ResolvedID = id
				st.ResolvedSource = source
				st.Versions = versions
				st.ResolveError = ""
				if id != "" && source != "" && source != "msstore" {
					cacheVersions(id, source, versions)
				}
			}
			updateVisibleCatalogItem(idx)
			updateCatalogFooter()
		})
	}()
}

func updateVisibleCatalogItem(idx int) {
	if currentPage != 1 {
		return
	}
	for slot, r := range rows {
		if r != nil && r.catalogIndex == idx {
			populateRowCombo(slot)
			procSendMessageW.Call(r.checkbox, BM_SETCHECK, boolCheck(states[idx].Selected), 0)
			return
		}
	}
}

func loadPackageData(app appDef) (string, string, []string, error) {

	for _, packageID := range catalogIDs(app) {
		versions, err := fetchVersionsFor(packageID, "winget")
		if err == nil {
			cachePackage(app.Name, packageID, "winget")
			return packageID, "winget", versions, nil
		}
	}

	if cached, ok := getCachedPackage(app.Name); ok {
		if cached.Source == "msstore" {
			ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
			_, err := runWinget(ctx, "show", "--id", cached.ID, "--exact", "--source", cached.Source, "--accept-source-agreements", "--disable-interactivity")
			cancel()
			if err == nil {
				return cached.ID, cached.Source, nil, nil
			}
			removeCachedPackage(app.Name)
		} else {
			versions, err := fetchVersionsFor(cached.ID, cached.Source)
			if err == nil {
				return cached.ID, cached.Source, versions, nil
			}
			removeCachedPackage(app.Name)
		}
	}

	id, source, err := resolvePackage(app.Name)
	if err == nil {
		cachePackage(app.Name, id, source)
		if source == "msstore" {
			return id, source, nil, nil
		}
		versions, verr := fetchVersionsFor(id, source)
		if verr == nil {
			return id, source, versions, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		_, showErr := runWinget(ctx, "show", "--id", id, "--exact", "--source", source, "--accept-source-agreements", "--disable-interactivity")
		cancel()
		if showErr == nil {
			return id, source, nil, nil
		}
	}

	return "", "", nil, fmt.Errorf("nem található ellenőrzött Winget/Microsoft Store csomag")
}

func resolvePackage(name string) (string, string, error) {
	if cached, ok := getCachedPackage(name); ok {
		return cached.ID, cached.Source, nil
	}
	queries := candidateQueries(name)
	sources := []string{"winget", "msstore"}

	for _, q := range queries {
		for _, source := range sources {
			ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
			out, err := runWinget(ctx, "search", "--name", q, "--exact", "--source", source, "--accept-source-agreements", "--disable-interactivity")
			cancel()
			if err != nil {
				continue
			}
			rows := parseSearchRows(out)
			matches, unique := exactSearchMatches(q, rows)
			if unique {
				cachePackage(name, matches[0].ID, source)
				return matches[0].ID, source, nil
			}
			if len(matches) > 1 {
				return "", "", fmt.Errorf("több pontos csomagegyezés található ehhez: %s", q)
			}
		}
	}

	return "", "", fmt.Errorf("nem található pontos Winget/Microsoft Store egyezés")
}

func parseSearchRows(out string) []searchRow {
	sep := false
	var rows []searchRow
	for _, raw := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		if isSeparatorLine(t) {
			sep = true
			continue
		}
		if !sep || t == "" {
			continue
		}
		cols := splitColsRe.Split(t, -1)
		if len(cols) < 2 {
			continue
		}
		id := strings.TrimSpace(cols[1])
		if id == "" || strings.Contains(strings.ToLower(id), "id") {
			continue
		}
		rows = append(rows, searchRow{Name: strings.TrimSpace(cols[0]), ID: id})
	}
	return rows
}

func parseInstalledPackages(out string) []installedPackage {
	sep := false
	var packages []installedPackage
	for _, raw := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		if isSeparatorLine(t) {
			sep = true
			continue
		}
		if !sep || t == "" {
			continue
		}
		cols := splitColsRe.Split(t, -1)
		if len(cols) < 3 {
			continue
		}
		name := strings.TrimSpace(cols[0])
		id := strings.TrimSpace(cols[1])
		version := strings.TrimSpace(cols[2])
		if name == "" || id == "" || version == "" {
			continue
		}
		source := ""
		if len(cols) >= 4 {
			last := strings.ToLower(strings.TrimSpace(cols[len(cols)-1]))
			if last == "winget" || last == "msstore" {
				source = last
			}
		}
		packages = append(packages, installedPackage{Name: name, ID: id, Version: version, Source: source})
	}
	return packages
}

func scanInstalledPrograms() {
	var packages []installedPackage
	if wingetOK {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		out, err := runWinget(ctx, "list", "--accept-source-agreements", "--disable-interactivity")
		cancel()
		if err == nil {
			packages = parseInstalledPackages(out)
		}
	}
	registryPackages := scanRegistryPackages()
	packages = enrichInstalledPackages(packages, registryPackages)
	uiDo(func() {
		applyInstalledPackages(packages, registryPackages)
		refreshFilter(true)
	})
}

func scanRegistryPackages() []registryPackage {
	queries := [][]string{
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "/reg:64", "machine"},
		{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "/reg:32", "machine"},
		{`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "/reg:64", "user"},
		{`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "/reg:32", "user"},
	}
	seen := map[string]bool{}
	var out []registryPackage
	for _, q := range queries {
		ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		cmd := exec.CommandContext(ctx, "reg.exe", "query", q[0], "/s", q[1])
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		data, err := cmd.CombinedOutput()
		cancel()
		if err != nil && len(data) == 0 {
			continue
		}
		for _, p := range parseRegistryQuery(string(data), q[2]) {
			key := strings.ToLower(strings.TrimSpace(p.RegistryKey))
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, p)
		}
	}
	return out
}

func parseRegistryQuery(text, scope string) []registryPackage {
	var out []registryPackage
	var cur *registryPackage
	flush := func() {
		if cur != nil && strings.TrimSpace(cur.DisplayName) != "" {
			out = append(out, *cur)
		}
		cur = nil
	}
	valueRe := regexp.MustCompile(`^\s+([^\s]+)\s+(REG_[A-Z0-9_]+)\s*(.*)$`)
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, "\r\n")
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trim), "HKEY_") {
			flush()
			cur = &registryPackage{RegistryKey: trim, Scope: scope}
			continue
		}
		if cur == nil || trim == "" {
			continue
		}
		m := valueRe.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(m[1]))
		value := strings.TrimSpace(m[3])
		switch name {
		case "displayname":
			cur.DisplayName = value
		case "displayversion":
			cur.DisplayVersion = value
		case "installlocation":
			cur.InstallLocation = value
		case "displayicon":
			cur.DisplayIcon = value
		case "uninstallstring":
			cur.UninstallString = value
		case "quietuninstallstring":
			cur.QuietUninstallString = value
		case "windowsinstaller":
			v := strings.ToLower(value)
			if v == "0x1" || v == "1" {
				cur.WindowsInstaller = 1
			}
		}
	}
	flush()
	return out
}

func enrichInstalledPackages(packages []installedPackage, registryPackages []registryPackage) []installedPackage {
	for i := range packages {
		if reg, ok := bestRegistryMatch(packages[i].Name, registryPackages); ok {
			packages[i].InstallLocation = deriveRegistryInstallLocation(reg)
			packages[i].DisplayIcon = reg.DisplayIcon
			packages[i].UninstallString = reg.UninstallString
			packages[i].QuietUninstallString = reg.QuietUninstallString
			packages[i].RegistryKey = reg.RegistryKey
			packages[i].Scope = reg.Scope
			packages[i].WindowsInstaller = reg.WindowsInstaller != 0
			if packages[i].Version == "" {
				packages[i].Version = reg.DisplayVersion
			}
		}
	}
	return packages
}

func bestRegistryMatch(name string, packages []registryPackage) (registryPackage, bool) {
	if normalizeSearch(name) == "" {
		return registryPackage{}, false
	}
	bestScore := 0
	best := registryPackage{}
	ambiguous := false
	for _, p := range packages {
		if normalizeSearch(p.DisplayName) == "" || !registryNamesCompatible(name, p.DisplayName) {
			continue
		}
		score := registryNameScore(name, p.DisplayName)
		if score > bestScore {
			bestScore = score
			best = p
			ambiguous = false
		} else if score > 0 && score == bestScore && !strings.EqualFold(strings.TrimSpace(best.RegistryKey), strings.TrimSpace(p.RegistryKey)) {
			ambiguous = true
		}
	}
	return best, bestScore >= 96 && !ambiguous
}

func deriveRegistryInstallLocation(p registryPackage) string {
	if v := cleanRegistryPath(p.InstallLocation); v != "" {
		return v
	}
	if v := executablePathFromRegistryValue(p.DisplayIcon); v != "" {
		if dir := safeExecutableDir(v); dir != "" {
			return dir
		}
	}
	for _, raw := range []string{p.QuietUninstallString, p.UninstallString} {
		if v := executablePathFromRegistryValue(raw); v != "" {
			if dir := safeExecutableDir(v); dir != "" {
				return dir
			}
		}
	}
	return ""
}

func cleanRegistryPath(v string) string {
	v = expandPercentEnv(strings.Trim(strings.TrimSpace(v), `"`))
	if v == "" {
		return ""
	}
	return filepath.Clean(v)
}

func expandPercentEnv(v string) string {
	re := regexp.MustCompile(`%([^%]+)%`)
	return re.ReplaceAllStringFunc(v, func(m string) string {
		parts := re.FindStringSubmatch(m)
		if len(parts) != 2 {
			return m
		}
		if e := os.Getenv(parts[1]); e != "" {
			return e
		}
		return m
	})
}

func executablePathFromRegistryValue(v string) string {
	v = expandPercentEnv(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	if i := strings.LastIndex(v, ","); i > 0 {
		tail := strings.TrimSpace(v[i+1:])
		if regexp.MustCompile(`^-?\d+$`).MatchString(tail) {
			v = strings.TrimSpace(v[:i])
		}
	}
	if strings.HasPrefix(v, `"`) {
		if i := strings.Index(v[1:], `"`); i >= 0 {
			return strings.TrimSpace(v[1 : i+1])
		}
	}
	lower := strings.ToLower(v)
	for _, ext := range []string{".exe", ".com", ".bat", ".cmd"} {
		if i := strings.Index(lower, ext); i >= 0 {
			return strings.Trim(strings.TrimSpace(v[:i+len(ext)]), `"`)
		}
	}
	return ""
}

func safeExecutableDir(exePath string) string {
	exePath = filepath.Clean(strings.TrimSpace(exePath))
	if exePath == "" {
		return ""
	}
	lower := strings.ToLower(exePath)
	if strings.Contains(lower, `\windows\installer\`) || strings.Contains(lower, `\programdata\package cache\`) || strings.Contains(lower, `\temp\`) {
		return ""
	}
	base := strings.ToLower(filepath.Base(exePath))
	if base == "msiexec.exe" || base == "rundll32.exe" {
		return ""
	}
	return filepath.Dir(exePath)
}

func applyInstalledPackages(packages []installedPackage, registryPackages []registryPackage) {
	byID := make(map[string]installedPackage, len(packages))
	byName := make(map[string][]installedPackage, len(packages))
	for _, pkg := range packages {
		if pkg.ID != "" {
			byID[strings.ToLower(strings.TrimSpace(pkg.ID))] = pkg
		}
		if k := normalizeSearch(pkg.Name); k != "" {
			byName[k] = append(byName[k], pkg)
		}
	}
	for i, app := range catalog {
		st := &states[i]
		st.Installed = false
		st.InstalledVersion = ""
		st.InstalledID = ""
		st.InstalledName = ""
		st.InstalledRegistryKey = ""
		st.InstalledDisplayIcon = ""
		st.ActualInstallDir = ""
		st.UninstallString = ""
		st.QuietUninstallString = ""
		st.InstallScope = ""
		st.WindowsInstaller = false
		st.CanUninstall = false
		st.InstalledMatchType = ""
		st.InstalledConfidence = 0
		st.UninstallStrategy = ""
		var pkg installedPackage
		found := false
		matchType := ""
		confidence := 0
		ids := append(catalogIDs(app), st.ResolvedID)
		for _, id := range ids {
			if id == "" {
				continue
			}
			if p, ok := byID[strings.ToLower(strings.TrimSpace(id))]; ok {
				pkg = p
				found = true
				matchType = "exact_package_id"
				confidence = 100
				break
			}
		}
		if !found {
			for _, q := range candidateQueries(app.Name) {
				matches := byName[normalizeSearch(q)]
				if len(matches) == 1 {
					pkg = matches[0]
					found = true
					matchType = "exact_display_alias"
					confidence = 96
					break
				}
			}
		}
		if !found {
			for _, q := range candidateQueries(app.Name) {
				if reg, ok := bestRegistryMatch(q, registryPackages); ok {
					pkg = installedPackage{Name: reg.DisplayName, Version: reg.DisplayVersion, InstallLocation: deriveRegistryInstallLocation(reg), DisplayIcon: reg.DisplayIcon, UninstallString: reg.UninstallString, QuietUninstallString: reg.QuietUninstallString, RegistryKey: reg.RegistryKey, Scope: reg.Scope, WindowsInstaller: reg.WindowsInstaller != 0}
					found = true
					matchType = "strict_registry_alias"
					confidence = 92
					break
				}
			}
		}
		if found {
			if pkg.RegistryKey == "" {
				if reg, ok := bestRegistryMatch(pkg.Name, registryPackages); ok {
					pkg.InstallLocation = deriveRegistryInstallLocation(reg)
					pkg.DisplayIcon = reg.DisplayIcon
					pkg.UninstallString = reg.UninstallString
					pkg.QuietUninstallString = reg.QuietUninstallString
					pkg.RegistryKey = reg.RegistryKey
					pkg.Scope = reg.Scope
					pkg.WindowsInstaller = reg.WindowsInstaller != 0
				}
			}
			strategy := installedUninstallStrategy(app, pkg)
			st.Installed = true
			st.InstalledVersion = pkg.Version
			st.InstalledID = pkg.ID
			st.InstalledName = pkg.Name
			st.InstalledRegistryKey = pkg.RegistryKey
			st.InstalledDisplayIcon = pkg.DisplayIcon
			st.ActualInstallDir = strings.TrimSpace(pkg.InstallLocation)
			st.UninstallString = pkg.UninstallString
			st.QuietUninstallString = pkg.QuietUninstallString
			st.InstallScope = pkg.Scope
			st.WindowsInstaller = pkg.WindowsInstaller
			st.InstalledMatchType = matchType
			st.InstalledConfidence = confidence
			st.UninstallStrategy = string(strategy)
			st.CanUninstall = strategyAllowsAutomaticUninstall(strategy)
			if strings.TrimSpace(pkg.InstallLocation) != "" {
				st.InstallDir = pkg.InstallLocation
			}
		}
	}
}

func uninstallCatalogProgram(idx int) {
	if idx < 0 || idx >= len(catalog) || !states[idx].Installed || !states[idx].CanUninstall {
		return
	}
	if isBusy() {
		messageBox(mainWnd, "Másik telepítési vagy eltávolítási művelet még folyamatban van.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	app := catalog[idx]
	versionText := ""
	if strings.TrimSpace(states[idx].InstalledVersion) != "" {
		versionText = "\nTelepített verzió: " + states[idx].InstalledVersion
	}
	if messageBox(mainWnd, "Eltávolítod ezt a programot?\n\n"+app.Name+versionText+"\n\nA Wiz4rd Fr0g automatikusan a megfelelő felhasználói vagy rendszergazdai jogosultsággal futtatja az eltávolítást.", appTitle, MB_YESNO|MB_ICONQUESTION) != IDYES {
		return
	}
	for _, r := range rows {
		if r != nil && r.catalogIndex == idx {
			procEnableWindow.Call(r.deleteBtn, 0)
			setText(r.deleteBtn, "Törlés...")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	setBusy(true, cancel)
	go func() {
		defer func() {
			cancel()
			setBusy(false, nil)
		}()
		uiDo(func() { appendLog("INFO", "Eltávolítás indítása: "+app.Name) })
		self, err := os.Executable()
		code := -1
		elevated := false
		logOffset := currentLogSize()
		if err == nil {
			code, elevated, err = executeUninstallWorkerFlow(ctx, self, idx)
		}
		if err == nil && code == uninstallCodeRunningProcess {
			approved := requestRunningProcessClose(idx)
			if approved {
				closeOK := false
				roots, exact := processRootsFromState(idx)
				if len(roots) == 0 && len(exact) == 0 {
					uiDo(func() {
						appendLog("WARN", app.Name+": nem található biztonságosan azonosítható futó folyamat; kézi bezárás után újrapróbálás.")
					})
					closeOK = true
				} else if elevated || strings.EqualFold(strings.TrimSpace(states[idx].InstallScope), "machine") {
					closeCode, closeErr := runElevated(ctx, self, []string{"--close-running-admin", strconv.Itoa(idx)})
					closeOK = closeErr == nil && closeCode == 0
					if closeErr != nil {
						err = closeErr
					}
				} else {
					closed, needElevation, closeErr := terminateKnownProcesses(roots, exact)
					if closeErr == nil && !needElevation {
						uiDo(func() { appendLog("INFO", fmt.Sprintf("%s: %d futó programfolyamat bezárva.", app.Name, closed)) })
						closeOK = true
					} else if needElevation {
						closeCode, elevateErr := runElevated(ctx, self, []string{"--close-running-admin", strconv.Itoa(idx)})
						closeOK = elevateErr == nil && closeCode == 0
						if elevateErr != nil {
							err = elevateErr
						}
					} else if closeErr != nil {
						err = closeErr
					}
				}
				if closeOK && err == nil {
					time.Sleep(500 * time.Millisecond)
					code, elevated, err = executeUninstallWorkerFlow(ctx, self, idx)
					_ = elevated
				}
			}
		}
		workerDetails := readLogSince(logOffset, 7000)
		uiDo(func() {
			for _, r := range rows {
				if r != nil && r.catalogIndex == idx {
					setText(r.deleteBtn, "Törlés")
					procEnableWindow.Call(r.deleteBtn, 1)
				}
			}
			if err != nil {
				messageBox(mainWnd, "Az eltávolítás indítása sikertelen:\n\n"+shorten(err.Error(), 900), appTitle, MB_OK|MB_ICONERROR)
				return
			}
			if code == uninstallCodeRebootRequired {
				appendLog("WARN", app.Name+": az eltávolítás végrehajtva, Windows újraindítás szükséges.")
				messageBox(mainWnd, app.Name+" eltávolítója újraindítást kér a művelet befejezéséhez.\n\nIndítsd újra a Windowst, majd a Wiz4rd Fr0g újraellenőrzi a telepített állapotot.", appTitle, MB_OK|MB_ICONWARNING)
				go scanInstalledPrograms()
				return
			}
			if code != uninstallCodeOK {
				detail := strings.TrimSpace(workerDetails)
				if detail == "" {
					detail = "Nem érkezett részletes worker napló."
				}
				if code == uninstallCodeUnsupported && isMicrosoftEdgeApp(app) {
					msg := "A Microsoft Edge ezen a gépen nem rendelkezik a Wiz4rd Fr0g által biztonságosan használható eltávolítási útvonallal.\n\nA WebView2 Runtime-ot a program nem kezeli Edge-ként és nem törli.\n\nMegnyitod a Windows Telepített alkalmazások oldalát?\n\nRészletek:\n" + shorten(detail, 1800)
					if messageBox(mainWnd, msg, appTitle, MB_YESNO|MB_ICONWARNING) == IDYES {
						openInstalledAppsSettings()
					}
				} else if code == uninstallCodeRunningProcess {
					messageBox(mainWnd, "Az eltávolító továbbra is futó programfolyamatot jelez.\n\nA Wiz4rd Fr0g csak a tényleges telepítési mappához biztonságosan köthető folyamatokat zárhatja be automatikusan.\n\nRészletek:\n"+shorten(detail, 1800), appTitle, MB_OK|MB_ICONWARNING)
				} else {
					messageBox(mainWnd, "A program eltávolítása sikertelen.\n\nKilépési kód: "+strconv.Itoa(code)+"\n\nRészletek:\n"+shorten(detail, 2200), appTitle, MB_OK|MB_ICONERROR)
				}
				go scanInstalledPrograms()
				return
			}
			states[idx].Installed = false
			states[idx].InstalledVersion = ""
			states[idx].InstalledID = ""
			states[idx].InstalledName = ""
			states[idx].InstalledRegistryKey = ""
			states[idx].InstalledDisplayIcon = ""
			states[idx].ActualInstallDir = ""
			states[idx].UninstallString = ""
			states[idx].QuietUninstallString = ""
			states[idx].InstallScope = ""
			states[idx].WindowsInstaller = false
			states[idx].CanUninstall = false
			states[idx].Selected = false
			states[idx].InstallDir = defaultInstallDir(app)
			appendLog("SUCCESS", app.Name+": eltávolítás befejeződött.")
			messageBox(mainWnd, app.Name+" eltávolítása befejeződött.", appTitle, MB_OK|MB_ICONINFO)
			refreshFilter(true)
			go scanInstalledPrograms()
		})
	}()
}

func executeUninstallWorkerFlow(ctx context.Context, self string, idx int) (int, bool, error) {
	app := catalog[idx]
	code, _, err := runDirectProcess(ctx, self, []string{"--uninstall-worker-user", strconv.Itoa(idx)})
	elevated := false
	if err == nil && code == uninstallCodeNeedElevation {
		uiDo(func() { appendLog("SYSTEM", app.Name+": gépszintű eltávolításhoz UAC szükséges.") })
		code, err = runElevated(ctx, self, []string{"--uninstall-worker-admin", strconv.Itoa(idx)})
		elevated = true
	}
	if err == nil && code == uninstallCodeWrongElevation {
		uiDo(func() {
			appendLog("INFO", app.Name+": felhasználói hatókör újrapróbálása normál jogosultsággal.")
		})
		code, _, err = runDirectProcess(ctx, self, []string{"--uninstall-worker-user", strconv.Itoa(idx)})
		elevated = false
	}
	return code, elevated, err
}

func requestRunningProcessClose(idx int) bool {
	answer := make(chan bool, 1)
	uiDo(func() {
		app := catalog[idx]
		roots, exact := processRootsFromState(idx)
		procs := enumerateMatchingProcesses(roots, exact)
		var b strings.Builder
		b.WriteString("Az eltávolító szerint a program még fut.\n\n")
		if len(procs) > 0 {
			b.WriteString("A Wiz4rd Fr0g az alábbi, tényleges telepítési útvonalhoz kötött folyamatokat találta:\n\n")
			limit := len(procs)
			if limit > 8 {
				limit = 8
			}
			for _, p := range procs[:limit] {
				b.WriteString(fmt.Sprintf("• %s (PID %d)\n", filepath.Base(p.Path), p.PID))
			}
			if len(procs) > limit {
				b.WriteString(fmt.Sprintf("• +%d további folyamat\n", len(procs)-limit))
			}
			b.WriteString("\nBezárja ezeket és újrapróbálja az eltávolítást?")
		} else {
			b.WriteString("Nem azonosítható olyan folyamat, amelyet biztonságosan automatikusan be lehet zárni.\n\nZárd be kézzel a programot, majd válaszd az Igen gombot az újrapróbáláshoz.")
		}
		answer <- messageBox(mainWnd, b.String(), appTitle+" - "+app.Name, MB_YESNO|MB_ICONWARNING) == IDYES
	})
	return <-answer
}

func runUninstallWorker(idx int, elevated bool) int {
	if idx < 0 || idx >= len(catalog) {
		workerLog("ERROR", "Érvénytelen katalógusindex az eltávolító workerben.")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	app := catalog[idx]
	mode := "felhasználói"
	if elevated {
		mode = "rendszergazdai"
	}
	workerLog("INFO", mode+" eltávolító worker elindult: "+app.Name)
	code := uninstallInContext(ctx, app, elevated)
	if code == uninstallCodeOK {
		workerLog("SUCCESS", app.Name+": eltávolítás sikeres.")
		return uninstallCodeOK
	}
	if code == uninstallCodeNeedElevation {
		workerLog("INFO", app.Name+": rendszergazdai jogosultság szükséges.")
		return uninstallCodeNeedElevation
	}
	if code == uninstallCodeWrongElevation {
		workerLog("INFO", app.Name+": az eltávolítást a bejelentkezett felhasználó kontextusában kell futtatni.")
		return uninstallCodeWrongElevation
	}
	if code == uninstallCodeRunningProcess {
		workerLog("WARN", app.Name+": az eltávolítást futó programfolyamat blokkolja.")
		return uninstallCodeRunningProcess
	}
	if code == uninstallCodeRebootRequired {
		workerLog("WARN", app.Name+": az eltávolítás újraindítást igényel.")
		return uninstallCodeRebootRequired
	}
	if code == uninstallCodeUnsupported {
		workerLog("WARN", app.Name+": nincs használható támogatott eltávolítási útvonal ezen a gépen.")
		return 20
	}
	workerLog("ERROR", app.Name+": az összes biztonságos eltávolítási módszer sikertelen.")
	return 1
}

func isMicrosoftEdgeApp(app appDef) bool {
	return strings.EqualFold(strings.TrimSpace(app.ID), "Microsoft.Edge") || normalizeSearch(app.Name) == "microsoftedge"
}

func openInstalledAppsSettings() {
	verb := utf16Ptr("open")
	target := utf16Ptr("ms-settings:appsfeatures")
	procShellExecuteW.Call(mainWnd, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, SW_SHOW)
}

func currentLogSize() int64 {
	path := filepath.Join(roamingAppData(), "Wiz4rdFr0g", "logs", "app.log")
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func readLogSince(offset int64, maxBytes int64) string {
	path := filepath.Join(roamingAppData(), "Wiz4rdFr0g", "logs", "app.log")
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() <= offset {
		return ""
	}
	start := offset
	if info.Size()-start > maxBytes {
		start = info.Size() - maxBytes
	}
	if _, err = f.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(f, maxBytes))
	if err != nil {
		return ""
	}
	return string(b)
}

func workerLog(level, text string) {
	line := fmt.Sprintf("[%s] [%s] %s\r\n", time.Now().Format("15:04:05"), level, strings.TrimSpace(text))
	writeLogFile(line)
}

func uninstallInContext(ctx context.Context, app appDef, elevated bool) int {
	if catalogBlocksAutomaticUninstall(app) {
		workerLog("WARN", app.Name+": a katalógusprofil nem enged automatikus eltávolítást.")
		return uninstallCodeUnsupported
	}
	registryPackages := scanRegistryPackages()
	userRegs := registryPackagesByScope(registryPackages, "user")
	machineRegs := registryPackagesByScope(registryPackages, "machine")
	userPackages := listInstalledPackagesForScope(ctx, "user")
	machinePackages := listInstalledPackagesForScope(ctx, "machine")

	if isMicrosoftEdgeApp(app) {
		if elevated {
			pkg, found := resolveExactWingetPackage(app, machinePackages)
			if !found {
				if _, userFound := resolveExactWingetPackage(app, userPackages); userFound {
					return uninstallCodeWrongElevation
				}
				return uninstallCodeUnsupported
			}
			pkg.Scope = "machine"
			return executeInstalledUninstall(ctx, app, pkg, machineRegs, true)
		}
		pkg, found := resolveExactWingetPackage(app, userPackages)
		if found {
			pkg.Scope = "user"
			return executeInstalledUninstall(ctx, app, pkg, userRegs, false)
		}
		if _, machineFound := resolveExactWingetPackage(app, machinePackages); machineFound {
			return uninstallCodeNeedElevation
		}
		return uninstallCodeUnsupported
	}

	if elevated {
		if _, userFound := resolveInstalledPackage(app, userPackages, userRegs); userFound {
			if _, machineFound := resolveInstalledPackage(app, machinePackages, machineRegs); !machineFound {
				return uninstallCodeWrongElevation
			}
		}
		pkg, found := resolveInstalledPackage(app, machinePackages, machineRegs)
		if found {
			pkg.Scope = "machine"
			return executeInstalledUninstall(ctx, app, pkg, machineRegs, true)
		}
		if !programStillInstalled(app) {
			return uninstallCodeOK
		}
		return uninstallCodeFailed
	}

	pkg, found := resolveInstalledPackage(app, userPackages, userRegs)
	if found {
		pkg.Scope = "user"
		return executeInstalledUninstall(ctx, app, pkg, userRegs, false)
	}
	if _, machineFound := resolveInstalledPackage(app, machinePackages, machineRegs); machineFound {
		return uninstallCodeNeedElevation
	}
	if !programStillInstalled(app) {
		return uninstallCodeOK
	}
	return uninstallCodeFailed
}

func registryPackagesByScope(packages []registryPackage, scope string) []registryPackage {
	out := make([]registryPackage, 0, len(packages))
	for _, p := range packages {
		if strings.EqualFold(strings.TrimSpace(p.Scope), scope) {
			out = append(out, p)
		}
	}
	return out
}

func listInstalledPackagesForScope(ctx context.Context, scope string) []installedPackage {
	if _, err := exec.LookPath("winget.exe"); err != nil {
		return nil
	}
	wctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	out, err := runWinget(wctx, "list", "--scope", scope, "--accept-source-agreements", "--disable-interactivity")
	if err != nil {
		return nil
	}
	packages := parseInstalledPackages(out)
	for i := range packages {
		packages[i].Scope = scope
	}
	return packages
}

func resolveExactWingetPackage(app appDef, packages []installedPackage) (installedPackage, bool) {
	for _, wantID := range catalogIDs(app) {
		for _, p := range packages {
			if strings.EqualFold(strings.TrimSpace(p.ID), strings.TrimSpace(wantID)) {
				return p, true
			}
		}
	}
	return installedPackage{}, false
}

func resolveInstalledPackage(app appDef, packages []installedPackage, registryPackages []registryPackage) (installedPackage, bool) {
	for _, wantID := range catalogIDs(app) {
		for _, p := range packages {
			if strings.EqualFold(strings.TrimSpace(p.ID), strings.TrimSpace(wantID)) {
				return p, true
			}
		}
	}
	for _, q := range candidateQueries(app.Name) {
		want := normalizeSearch(q)
		var matches []installedPackage
		seenIDs := map[string]bool{}
		for _, p := range packages {
			if normalizeSearch(p.Name) != want {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(p.ID))
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(p.Name)) + "|" + strings.ToLower(strings.TrimSpace(p.Version))
			}
			if !seenIDs[key] {
				seenIDs[key] = true
				matches = append(matches, p)
			}
		}
		if len(matches) == 1 {
			return matches[0], true
		}
		if len(matches) > 1 {
			return installedPackage{}, false
		}
	}
	for _, q := range candidateQueries(app.Name) {
		if reg, ok := bestRegistryMatch(q, registryPackages); ok {
			return installedPackage{Name: reg.DisplayName, Version: reg.DisplayVersion, InstallLocation: deriveRegistryInstallLocation(reg), UninstallString: reg.UninstallString, QuietUninstallString: reg.QuietUninstallString, RegistryKey: reg.RegistryKey, Scope: reg.Scope, WindowsInstaller: reg.WindowsInstaller != 0}, true
		}
	}
	return installedPackage{}, false
}

func resolveRegistryForInstalled(app appDef, pkg installedPackage, registryPackages []registryPackage) (registryPackage, bool) {
	if strings.TrimSpace(pkg.Name) != "" {
		if reg, ok := bestRegistryMatch(pkg.Name, registryPackages); ok {
			return reg, true
		}
	}
	for _, q := range candidateQueries(app.Name) {
		if reg, ok := bestRegistryMatch(q, registryPackages); ok {
			return reg, true
		}
	}
	return registryPackage{}, false
}

func runRegisteredUninstaller(ctx context.Context, app appDef, reg registryPackage) uninstallAttempt {
	raw := strings.TrimSpace(reg.QuietUninstallString)
	quiet := raw != ""
	if raw == "" {
		raw = strings.TrimSpace(reg.UninstallString)
	}
	if raw == "" {
		return uninstallAttempt{ExitCode: -1, Err: fmt.Errorf("hiányzó uninstall parancs")}
	}
	if guid := extractMSIProductCode(raw + " " + reg.RegistryKey); guid != "" || reg.WindowsInstaller != 0 {
		if guid == "" {
			workerLog("WARN", app.Name+": MSI bejegyzéshez nem található ProductCode.")
			return uninstallAttempt{ExitCode: -1, Err: fmt.Errorf("MSI ProductCode nem található")}
		}
		args := []string{"/x", guid, "/qn", "/norestart"}
		workerLog("SYSTEM", "MSI eltávolítás: "+formatCommand("msiexec.exe", args))
		code, out, err := runDirectProcess(ctx, "msiexec.exe", args)
		attempt := makeUninstallAttempt(code, out, err)
		if attempt.Success {
			return attempt
		}
		workerLog("WARN", app.Name+": csendes MSI eltávolítás sikertelen: "+compactFailure(out, err)+fmt.Sprintf(" (exit=%d)", code))
		args = []string{"/x", guid, "/norestart"}
		code2, out2, err2 := runDirectProcess(ctx, "msiexec.exe", args)
		attempt2 := makeUninstallAttempt(code2, strings.TrimSpace(out+"\n"+out2), err2)
		if attempt2.Success {
			return attempt2
		}
		if code2 == 1605 && !programStillInstalled(app) {
			attempt2.Success = true
			return attempt2
		}
		workerLog("WARN", app.Name+": MSI eltávolítás sikertelen: "+compactFailure(out2, err2)+fmt.Sprintf(" (exit=%d)", code2))
		return attempt2
	}
	exe, args, err := splitRegisteredCommand(raw)
	if err != nil {
		workerLog("WARN", app.Name+": Registry UninstallString nem dolgozható fel: "+err.Error())
		return uninstallAttempt{ExitCode: -1, Err: err, Output: err.Error()}
	}
	if !quiet {
		lower := strings.ToLower(filepath.Base(exe))
		joined := strings.ToLower(strings.Join(args, " "))
		if regexp.MustCompile(`unins\d*\.exe`).MatchString(lower) && !strings.Contains(joined, "/verysilent") {
			args = append(args, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")
		}
	}
	workerLog("SYSTEM", "Regisztrált eltávolító: "+formatCommand(exe, args))
	code, out, err := runDirectProcess(ctx, exe, args)
	attempt := makeUninstallAttempt(code, out, err)
	if !attempt.Success {
		workerLog("WARN", app.Name+": regisztrált eltávolító sikertelen: "+compactFailure(out, err)+fmt.Sprintf(" (exit=%d)", code))
	}
	return attempt
}

func runWingetUninstallScoped(ctx context.Context, app appDef, pkg installedPackage, found bool, scope string) uninstallAttempt {
	if _, err := exec.LookPath("winget.exe"); err != nil {
		return uninstallAttempt{ExitCode: -1, Err: err, Output: err.Error()}
	}
	args := []string{"uninstall"}
	if found && strings.TrimSpace(pkg.ID) != "" {
		args = append(args, "--id", pkg.ID, "--exact")
	} else if ids := catalogIDs(app); len(ids) > 0 {
		args = append(args, "--id", ids[0], "--exact")
	} else if found && strings.TrimSpace(pkg.Name) != "" {
		args = append(args, "--name", pkg.Name, "--exact")
	} else {
		args = append(args, "--name", app.Name, "--exact")
	}
	if scope != "" {
		args = append(args, "--scope", scope)
	}
	if strings.EqualFold(strings.TrimSpace(pkg.Source), "msstore") {
		args = append(args, "--source", "msstore")
	}
	args = append(args, "--all-versions", "--silent", "--accept-source-agreements", "--disable-interactivity")
	workerLog("WINGET", formatCommand("winget.exe", args))
	out, err := runWinget(ctx, args...)
	if err == nil {
		return makeUninstallAttempt(0, out, nil)
	}
	code := -1
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	}
	if uninstallExitSuccess(code) {
		return makeUninstallAttempt(code, out, err)
	}
	workerLog("WARN", app.Name+": Winget csendes eltávolítás sikertelen: "+compactFailure(out, err))
	if uninstallWrongElevation(out) || uninstallNeedsElevation(out) || uninstallRunningProcess(out) {
		return makeUninstallAttempt(code, out, err)
	}
	args2 := make([]string, 0, len(args))
	for _, a := range args {
		if a != "--silent" && a != "--disable-interactivity" {
			args2 = append(args2, a)
		}
	}
	workerLog("WINGET", "Interaktív fallback: "+formatCommand("winget.exe", args2))
	out2, err2 := runWinget(ctx, args2...)
	if err2 == nil {
		return makeUninstallAttempt(0, strings.TrimSpace(out+"\n"+out2), nil)
	}
	code2 := -1
	if ee, ok := err2.(*exec.ExitError); ok {
		code2 = ee.ExitCode()
	}
	if uninstallExitSuccess(code2) {
		return makeUninstallAttempt(code2, strings.TrimSpace(out+"\n"+out2), err2)
	}
	workerLog("WARN", app.Name+": Winget fallback sikertelen: "+compactFailure(out2, err2))
	return makeUninstallAttempt(code2, strings.TrimSpace(out+"\n"+out2), err2)
}

func splitRegisteredCommand(raw string) (string, []string, error) {
	raw = expandPercentEnv(strings.TrimSpace(raw))
	exe, args, err := splitRegisteredCommandRaw(raw)
	if err != nil {
		return "", nil, err
	}
	return resolveExecutable(exe, args)
}

func resolveExecutable(exe string, args []string) (string, []string, error) {
	exe = strings.Trim(strings.TrimSpace(exe), `"`)
	if exe == "" {
		return "", nil, fmt.Errorf("hiányzó futtatható fájl")
	}
	if p, err := exec.LookPath(exe); err == nil {
		exe = p
	} else if filepath.IsAbs(exe) {
		if _, statErr := os.Stat(exe); statErr != nil {
			return "", nil, fmt.Errorf("az eltávolító nem található: %s", exe)
		}
	}
	return exe, args, nil
}

func runDirectProcess(ctx context.Context, exe string, args []string) (int, string, error) {
	if ext := strings.ToLower(filepath.Ext(exe)); ext == ".cmd" || ext == ".bat" {
		comspec := os.Getenv("ComSpec")
		if comspec == "" {
			comspec = `C:\Windows\System32\cmd.exe`
		}
		cmdArgs := append([]string{"/D", "/S", "/C", exe}, args...)
		return runDirectProcess(ctx, comspec, cmdArgs)
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	if filepath.IsAbs(exe) {
		cmd.Dir = filepath.Dir(exe)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	data, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return -1, string(data), ctx.Err()
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), string(data), err
		}
		return -1, string(data), err
	}
	return 0, string(data), nil
}

func verifyProgramRemoved(app appDef) bool {
	for i := 0; i < 8; i++ {
		if !programStillInstalled(app) {
			return true
		}
		time.Sleep(700 * time.Millisecond)
	}
	return !programStillInstalled(app)
}

func programStillInstalled(app appDef) bool {
	var wingetPackages []installedPackage
	if _, err := exec.LookPath("winget.exe"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		out, runErr := runWinget(ctx, "list", "--accept-source-agreements", "--disable-interactivity")
		cancel()
		if runErr == nil {
			wingetPackages = parseInstalledPackages(out)
		}
	}
	regs := scanRegistryPackages()
	_, found := resolveInstalledPackage(app, wingetPackages, regs)
	return found
}

func isProgramFilesPath(path string) bool {
	path = strings.ToLower(filepath.Clean(strings.TrimSpace(path)))
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		root := strings.TrimSpace(os.Getenv(env))
		if root != "" && strings.HasPrefix(path, strings.ToLower(filepath.Clean(root))+`\`) {
			return true
		}
	}
	return false
}

func runElevated(ctx context.Context, exe string, args []string) (int, error) {
	if p, err := exec.LookPath(exe); err == nil {
		exe = p
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString(windowsArgumentLine(args))
	info := shellExecuteInfo{
		fMask:        SEE_MASK_NOCLOSEPROCESS,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: params,
		nShow:        SW_SHOWNORMAL,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	uiDo(func() { appendLog("SYSTEM", "UAC / emelt futtatás: "+formatCommand(exe, args)) })
	r1, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return -1, fmt.Errorf("rendszergazdai indítás sikertelen: %v", callErr)
	}
	if info.hProcess == 0 {
		return -1, fmt.Errorf("az emelt folyamat nem adott process handle-t")
	}
	defer procCloseHandle.Call(info.hProcess)
	for {
		select {
		case <-ctx.Done():
			procTerminateProcess.Call(info.hProcess, 1)
			return -1, ctx.Err()
		default:
		}
		wait, _, _ := procWaitForSingleObject.Call(info.hProcess, 200)
		if wait == WAIT_TIMEOUT {
			continue
		}
		if wait != WAIT_OBJECT_0 {
			return -1, fmt.Errorf("várakozási hiba: 0x%X", wait)
		}
		var code uint32
		r, _, err := procGetExitCodeProcess.Call(info.hProcess, uintptr(unsafe.Pointer(&code)))
		if r == 0 {
			return -1, fmt.Errorf("kilépési kód lekérése sikertelen: %v", err)
		}
		return int(code), nil
	}
}

func windowsArgumentLine(args []string) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, quoteWindowsArg(a))
	}
	return strings.Join(parts, " ")
}

func quoteWindowsArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range s {
		if r == '\\' {
			slashes++
			continue
		}
		if r == '"' {
			b.WriteString(strings.Repeat(`\\`, slashes*2+1))
			b.WriteRune(r)
			slashes = 0
			continue
		}
		b.WriteString(strings.Repeat(`\\`, slashes))
		slashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat(`\\`, slashes*2))
	b.WriteByte('"')
	return b.String()
}

func psQuote(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

func compactFailure(out string, err error) string {
	msg := strings.TrimSpace(out)
	if msg == "" && err != nil {
		msg = err.Error()
	}
	return shorten(msg, 500)
}

func redactCommand(v string) string {
	return shorten(strings.TrimSpace(v), 500)
}

func fetchVersionsFor(id, source string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	out, err := runWinget(ctx, "show", "--id", id, "--exact", "--versions", "--source", source, "--accept-source-agreements", "--disable-interactivity")
	if err != nil {
		return nil, err
	}
	return parseVersions(out), nil
}

func goStep2() {
	if isBusy() {
		messageBox(mainWnd, "Az előző művelet megszakítása még folyamatban van. Várj néhány másodpercet, majd próbáld újra.", appTitle, MB_OK|MB_ICONINFO)
		return
	}
	dir := strings.TrimSpace(getText(downloadDirEdit))
	if dir == "" {
		messageBox(mainWnd, "A letöltési mappa nem lehet üres.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		messageBox(mainWnd, "A letöltési mappa nem hozható létre:\n"+err.Error(), appTitle, MB_OK|MB_ICONERROR)
		return
	}
	selected = nil
	var pending, duplicates []string
	seenPackages := map[string]bool{}
	for i, st := range states {
		if !st.Selected {
			continue
		}
		if st.VersionsLoading || !st.VersionsLoaded {
			pending = append(pending, catalog[i].Name)
			continue
		}
		if st.ResolveError != "" || st.ResolvedID == "" || st.ResolvedSource == "" {
			pending = append(pending, catalog[i].Name+" (nincs ellenőrzött csomag)")
			continue
		}
		packageKey := strings.ToLower(st.ResolvedSource + "|" + st.ResolvedID)
		if seenPackages[packageKey] {
			duplicates = append(duplicates, catalog[i].Name)
			continue
		}
		seenPackages[packageKey] = true
		source := st.ResolvedSource
		selected = append(selected, selection{
			CatalogIndex:     i,
			ID:               st.ResolvedID,
			Source:           source,
			Name:             catalog[i].Name,
			Version:          st.Version,
			InstallDir:       strings.TrimSpace(st.InstallDir),
			CustomInstallDir: !samePath(strings.TrimSpace(st.InstallDir), defaultInstallDir(catalog[i])),
		})
	}
	if len(pending) > 0 {
		messageBox(mainWnd, "Néhány kijelölt program csomagadata még betöltés alatt van:\n\n"+strings.Join(firstN(pending, 10), "\n"), appTitle, MB_OK|MB_ICONINFO)
		return
	}
	if len(selected) == 0 {
		messageBox(mainWnd, "Jelölj ki legalább egy telepíthető programot.", appTitle, MB_OK|MB_ICONINFO)
		return
	}
	if len(duplicates) > 0 {
		messageBox(mainWnd, "Néhány kijelölt katalógusbejegyzés ugyanarra a csomagra mutat. Ezeket csak egyszer telepítjük:\n\n"+strings.Join(firstN(duplicates, 10), "\n"), appTitle, MB_OK|MB_ICONINFO)
	}
	saveDownloadDir(dir)
	buildSummary()
	setPage(2)
}

func buildSummary() {
	statusByIndex = make(map[int]string)
	for _, it := range selected {
		statusByIndex[it.CatalogIndex] = "Telepítésre vár"
	}
	refreshSummaryText()
	setText(operationLabel, fmt.Sprintf("%d program telepítésre vár.", len(selected)))
	procEnableWindow.Call(installBtn, 1)
	procEnableWindow.Call(back2Btn, 1)
	procEnableWindow.Call(next2Btn, 0)
	clearLog()
}

func refreshSummaryText() {
	if summaryEdit == 0 {
		return
	}
	var b strings.Builder
	for i, it := range selected {
		v := it.Version
		if v == "" {
			v = "Legújabb"
		}
		loc := it.InstallDir
		if loc == "" {
			loc = "Alapértelmezett"
		}
		status := statusByIndex[it.CatalogIndex]
		if status == "" {
			status = "Telepítésre vár"
		}
		fmt.Fprintf(&b, "%02d. [%s] %s\r\n    Verzió: %s | Forrás: %s\r\n    Hely: %s\r\n\r\n", i+1, status, it.Name, v, it.Source, loc)
	}
	setText(summaryEdit, b.String())
}

func setSummaryStatus(idx int, text string) {
	if statusByIndex == nil {
		statusByIndex = map[int]string{}
	}
	statusByIndex[idx] = text
	refreshSummaryText()
}

func downloadedArtifactExists(root string) bool {
	found := false
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || info.Size() <= 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		switch ext {
		case ".yaml", ".yml", ".json", ".txt", ".log":
			return nil
		}
		found = true
		return io.EOF
	})
	return found
}

func verifyProgramInstalled(app appDef) bool {
	for i := 0; i < 8; i++ {
		var wingetPackages []installedPackage
		if _, err := exec.LookPath("winget.exe"); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			out, runErr := runWinget(ctx, "list", "--accept-source-agreements", "--disable-interactivity")
			cancel()
			if runErr == nil {
				wingetPackages = parseInstalledPackages(out)
			}
		}
		regs := scanRegistryPackages()
		if _, found := resolveInstalledPackage(app, wingetPackages, regs); found {
			return true
		}
		time.Sleep(850 * time.Millisecond)
	}
	return false
}

func startInstall() {
	if isBusy() || len(selected) == 0 {
		return
	}
	if !wingetOK {
		messageBox(mainWnd, "A winget nem elérhető, ezért a telepítés nem indítható.", appTitle, MB_OK|MB_ICONERROR)
		return
	}
	results = make(map[int]*resultInfo)
	for _, it := range selected {
		results[it.CatalogIndex] = &resultInfo{}
	}
	dlRoot := strings.TrimSpace(getText(downloadDirEdit))
	ctx, cancel := context.WithCancel(context.Background())
	setBusy(true, cancel)
	procEnableWindow.Call(installBtn, 0)
	procEnableWindow.Call(cancelBtn, 1)
	procEnableWindow.Call(next2Btn, 0)
	appendLog("INFO", fmt.Sprintf("Telepítési folyamat elindult. Programok: %d", len(selected)))
	go func() {
		defer func() {
			uiDo(func() {
				setBusy(false, nil)
				procEnableWindow.Call(back2Btn, 1)
				procEnableWindow.Call(cancelBtn, 0)
				procEnableWindow.Call(next2Btn, 1)
				setText(operationLabel, "A folyamat befejeződött.")
			})
			go scanInstalledPrograms()
		}()
		for pos, it := range selected {
			if ctx.Err() != nil {
				uiDo(func() { setSummaryStatus(it.CatalogIndex, "Megszakítva") })
				break
			}
			uiDo(func() {
				setText(operationLabel, fmt.Sprintf("%d/%d: %s", pos+1, len(selected), it.Name))
				setSummaryStatus(it.CatalogIndex, "Csomag feloldása...")
			})

			resolvedID, resolvedSource := it.ID, it.Source
			if resolvedID == "" || resolvedSource == "" {
				res := results[it.CatalogIndex]
				res.Error = "nincs ellenőrzött csomagazonosító"
				uiDo(func() {
					setSummaryStatus(it.CatalogIndex, "Hiba")
					appendLog("ERROR", it.Name+": nincs ellenőrzött csomagazonosító.")
				})
				continue
			}

			uiDo(func() { setSummaryStatus(it.CatalogIndex, "Letöltés...") })
			res := results[it.CatalogIndex]
			target := filepath.Join(dlRoot, safeName(it.Name), versionFolder(it.Version))
			_ = os.MkdirAll(target, 0755)

			downloadOK := false
			if resolvedSource == "msstore" {
				res.DownloadSkipped = true
				uiDo(func() {
					appendLog("WARN", it.Name+": Microsoft Store csomaghoz külön installer nem menthető; közvetlen telepítés következik.")
				})
			} else if resolvedID != "" && resolvedSource == "winget" {
				dargs := []string{"download", "--id", resolvedID, "--exact", "--source", "winget", "--download-directory", target, "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
				if it.Version != "" {
					dargs = append(dargs, "--version", it.Version)
				}
				code, err := runStreaming(ctx, "winget.exe", dargs...)
				downloadOK = err == nil && code == 0 && downloadedArtifactExists(target)
				if err == nil && code == 0 && !downloadOK {
					uiDo(func() {
						appendLog("WARN", it.Name+": a Winget sikeres kódot adott, de a célmappában nem található ellenőrizhető telepítőcsomag.")
					})
				}
			} else {
				res.Error = "nincs támogatott, ellenőrzött letöltési forrás"
			}
			if downloadOK {
				res.DownloadOK = true
				uiDo(func() { appendLog("SUCCESS", it.Name+": telepítőcsomag letöltve.") })
			} else if !res.DownloadSkipped {
				res.Error = "telepítőcsomag külön letöltése nem sikerült"
				uiDo(func() {
					appendLog("WARN", it.Name+": a telepítőcsomag mentése nem sikerült; közvetlen telepítést még megpróbáljuk.")
				})
			}

			if ctx.Err() != nil {
				uiDo(func() { setSummaryStatus(it.CatalogIndex, "Megszakítva") })
				break
			}
			uiDo(func() { setSummaryStatus(it.CatalogIndex, "Telepítés...") })

			installOK := false
			if resolvedID != "" && (resolvedSource == "winget" || resolvedSource == "msstore") {
				iargs := []string{"install", "--id", resolvedID, "--exact", "--source", resolvedSource, "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
				if it.Version != "" && resolvedSource == "winget" {
					iargs = append(iargs, "--version", it.Version)
				}
				if it.CustomInstallDir && it.InstallDir != "" && resolvedSource == "winget" {
					iargs = append(iargs, "--location", it.InstallDir)
					uiDo(func() {
						appendLog("INFO", it.Name+": kért telepítési hely: "+it.InstallDir+" (ha a telepítő támogatja)")
					})
				} else if it.CustomInstallDir && resolvedSource == "msstore" {
					uiDo(func() {
						appendLog("WARN", it.Name+": az egyedi telepítési hely Microsoft Store csomagnál nem alkalmazható.")
					})
				}
				code, err := runStreaming(ctx, "winget.exe", iargs...)
				installOK = err == nil && code == 0
				if installOK {
					installOK = verifyProgramInstalled(catalog[it.CatalogIndex])
					if !installOK {
						uiDo(func() {
							appendLog("ERROR", it.Name+": a telepítő sikeres kódot adott, de a program telepített állapota nem igazolható.")
						})
					}
				}
			} else {
				installOK = false
			}

			if installOK {
				res.InstallOK = true
				uiDo(func() {
					setSummaryStatus(it.CatalogIndex, "Kész")
					appendLog("SUCCESS", it.Name+": telepítés sikeres.")
				})
			} else {
				if res.Error != "" {
					res.Error += "; "
				}
				res.Error += "telepítés minden biztonságos Winget/Microsoft Store feloldással sikertelen"
				uiDo(func() {
					if ctx.Err() != nil {
						setSummaryStatus(it.CatalogIndex, "Megszakítva")
					} else {
						setSummaryStatus(it.CatalogIndex, "Hiba")
					}
					appendLog("ERROR", it.Name+": nem található/telepíthető biztonságosan egyik támogatott csomagforrásból sem.")
				})
			}
		}
		uiDo(func() { appendLog("INFO", "Telepítési folyamat befejeződött.") })
	}()
}

func runStreaming(ctx context.Context, exe string, args ...string) (int, error) {
	uiDo(func() { appendLog("CMD", formatCommand(exe, args)) })
	cmd := exec.CommandContext(ctx, exe, args...)
	if filepath.IsAbs(exe) {
		cmd.Dir = filepath.Dir(exe)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdout, e := cmd.StdoutPipe()
	if e != nil {
		return -1, e
	}
	stderr, e := cmd.StderrPipe()
	if e != nil {
		return -1, e
	}
	if e = cmd.Start(); e != nil {
		return -1, e
	}
	var wg sync.WaitGroup
	scan := func(r io.Reader, level string) {
		defer wg.Done()
		s := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		s.Buffer(buf, 1024*1024)
		for s.Scan() {
			line := strings.TrimSpace(strings.TrimRight(s.Text(), "\r"))
			if line != "" {
				l := line
				uiDo(func() { appendLog(level, l) })
			}
		}
	}
	wg.Add(2)
	go scan(stdout, "OUT")
	go scan(stderr, "STDERR")
	err := cmd.Wait()
	wg.Wait()
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), err
		}
		return -1, err
	}
	return 0, nil
}

func cancelOperation() {
	stateMu.Lock()
	c := currentCancel
	stateMu.Unlock()
	if c != nil {
		setText(operationLabel, "Megszakítás folyamatban...")
		appendLog("WARN", "Megszakítás kérve.")
		c()
	}
}
func setBusy(v bool, c context.CancelFunc) {
	stateMu.Lock()
	busy = v
	currentCancel = c
	stateMu.Unlock()
}
func isBusy() bool { stateMu.Lock(); defer stateMu.Unlock(); return busy }

func runWinget(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "winget.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	data, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(data), ctx.Err()
	}
	return string(data), err
}

func parseVersions(out string) []string {
	seen := map[string]bool{}
	var vs []string
	sep := false
	for _, raw := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		if isSeparatorLine(t) {
			sep = true
			continue
		}
		if !sep || t == "" {
			continue
		}
		f := strings.Fields(t)
		if len(f) == 0 {
			continue
		}
		v := f[0]
		if versionRegexp.MatchString(v) && containsDigit(v) && !seen[v] {
			seen[v] = true
			vs = append(vs, v)
		}
	}
	return vs
}

func checkWinget() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	_, err := runWinget(ctx, "--version")
	cancel()
	uiDo(func() {
		wingetOK = err == nil
		if wingetOK {
			bindCatalogPage()
		} else {
			for slot := range rows {
				populateRowCombo(slot)
			}
			messageBox(mainWnd, "A Winget nem érhető el. A katalógus böngészhető, a telepített programokat a Windows rendszerleíró adatbázisából továbbra is felismerjük.", appTitle, MB_OK|MB_ICONWARNING)
		}
		go scanInstalledPrograms()
	})
}

func refreshWingetSourcesAsync() {
	sourceRefreshOnce.Do(func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			_, _ = runWinget(ctx, "source", "update", "--disable-interactivity")
			cancel()
		}()
	})
}

func buildFinish() {
	ok, fail := 0, 0
	var b strings.Builder
	b.WriteString("Telepítés befejezve\r\n\r\n")
	for _, it := range selected {
		r := results[it.CatalogIndex]
		status := "Megszakítva"
		if r != nil && r.InstallOK {
			status = "Sikeres"
			ok++
		} else {
			fail++
		}
		v := it.Version
		if v == "" {
			v = "Legújabb"
		}
		fmt.Fprintf(&b, "%s | %s | %s\r\n", it.Name, v, status)
		if r != nil && r.Error != "" {
			fmt.Fprintf(&b, "    %s\r\n", r.Error)
		}
	}
	fmt.Fprintf(&b, "\r\nSikeres: %d | Sikertelen/megszakított: %d\r\n", ok, fail)
	setText(finishSummaryEdit, b.String())
	scanSetupMeta()
}

func scanSetupMeta() {
	procEnableWindow.Call(deleteSetupBtn, 0)
	meta, err := readInstallerMetadata()
	if err != nil {
		setText(cleanupInfo, "Nincs nyilvántartott eredeti Wiz4rd_Fr0g_Setup.exe.")
		return
	}
	if _, err = os.Stat(meta.SourcePath); err != nil {
		setText(cleanupInfo, "Az eredeti Setup.exe már nem található.")
		return
	}
	h, err := fileSHA256(meta.SourcePath)
	if err != nil || !strings.EqualFold(h, meta.SHA256) {
		setText(cleanupInfo, "A Setup.exe megváltozott; biztonsági okból nem törölhető.")
		return
	}
	setText(cleanupInfo, "Törölhető: "+meta.SourcePath)
	procEnableWindow.Call(deleteSetupBtn, 1)
}

func deleteSetup() {
	meta, err := readInstallerMetadata()
	if err != nil {
		return
	}
	h, err := fileSHA256(meta.SourcePath)
	if err != nil || !strings.EqualFold(h, meta.SHA256) {
		messageBox(mainWnd, "A Setup.exe nem egyezik a telepítéskor rögzített fájllal.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	if messageBox(mainWnd, "Biztosan törlöd az eredeti Setup.exe fájlt?\n\n"+meta.SourcePath, appTitle, MB_YESNO|MB_ICONQUESTION) != IDYES {
		return
	}
	if err := os.Remove(meta.SourcePath); err != nil {
		messageBox(mainWnd, "A Setup.exe nem törölhető:\n"+err.Error(), appTitle, MB_OK|MB_ICONERROR)
		return
	}
	_ = os.Remove(metadataPath())
	setText(cleanupInfo, "Az eredeti Setup.exe törölve.")
	procEnableWindow.Call(deleteSetupBtn, 0)
}

func uiDo(fn func()) {
	uiMu.Lock()
	uiQueue = append(uiQueue, fn)
	uiMu.Unlock()
	if mainWnd != 0 {
		procPostMessageW.Call(mainWnd, WM_APP_UI, 0, 0)
	}
}
func drainUITasks() {
	uiMu.Lock()
	tasks := uiQueue
	uiQueue = nil
	uiMu.Unlock()
	for _, fn := range tasks {
		fn()
	}
}

func appendLog(level, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	line := fmt.Sprintf("[%s] [%s] %s\r\n", time.Now().Format("15:04:05"), level, strings.TrimSpace(text))
	writeLogFile(line)
	if logEdit != 0 {
		l, _, _ := procGetWindowTextLenW.Call(logEdit)
		procSendMessageW.Call(logEdit, EM_SETSEL, l, l)
		p := utf16Ptr(line)
		procSendMessageW.Call(logEdit, EM_REPLACESEL, 0, uintptr(unsafe.Pointer(p)))
	}
}
func clearLog() {
	if logEdit != 0 {
		setText(logEdit, "")
	}
}
func writeLogFile(line string) {
	dir := filepath.Join(roamingAppData(), "Wiz4rdFr0g", "logs")
	_ = os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		_, _ = f.WriteString(line)
	}
}

func defaultInstallDir(app appDef) string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	pf86 := os.Getenv("ProgramFiles(x86)")
	if pf86 == "" {
		pf86 = `C:\Program Files (x86)`
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		home, _ := os.UserHomeDir()
		local = filepath.Join(home, "AppData", "Local")
	}
	roaming := os.Getenv("APPDATA")
	if roaming == "" {
		home, _ := os.UserHomeDir()
		roaming = filepath.Join(home, "AppData", "Roaming")
	}
	id := app.ID
	switch id {
	case "Google.Chrome":
		return filepath.Join(pf, "Google", "Chrome", "Application")
	case "Mozilla.Firefox":
		return filepath.Join(pf, "Mozilla Firefox")
	case "Brave.Brave":
		return filepath.Join(pf, "BraveSoftware", "Brave-Browser", "Application")
	case "7zip.7zip":
		return filepath.Join(pf, "7-Zip")
	case "VideoLAN.VLC":
		return filepath.Join(pf, "VideoLAN", "VLC")
	case "Microsoft.VisualStudioCode":
		return filepath.Join(local, "Programs", "Microsoft VS Code")
	case "Git.Git":
		return filepath.Join(pf, "Git")
	case "Python.Python.3.13":
		return filepath.Join(local, "Programs", "Python", "Python313")
	case "OpenJS.NodeJS.LTS":
		return filepath.Join(pf, "nodejs")
	case "Notepad++.Notepad++":
		return filepath.Join(pf, "Notepad++")
	case "OBSProject.OBSStudio":
		return filepath.Join(pf, "obs-studio")
	case "Discord.Discord":
		return filepath.Join(local, "Discord")
	case "Spotify.Spotify":
		return filepath.Join(roaming, "Spotify")
	case "Valve.Steam":
		return filepath.Join(pf86, "Steam")
	case "RARLab.WinRAR":
		return filepath.Join(pf, "WinRAR")
	case "Microsoft.PowerToys":
		return filepath.Join(pf, "PowerToys")
	default:
		return filepath.Join(pf, safeFolderName(app.Name))
	}
}

func samePath(a, b string) bool {
	a = strings.TrimRight(strings.TrimSpace(a), `\\/`)
	b = strings.TrimRight(strings.TrimSpace(b), `\\/`)
	return strings.EqualFold(a, b)
}

func defaultDownloadDir() string {
	if s := loadDownloadDir(); s != "" {
		return s
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "Wiz4rd Fr0g")
}
func settingsPath() string { return filepath.Join(roamingAppData(), "Wiz4rdFr0g", "settings.json") }
func saveDownloadDir(dir string) {
	_ = os.MkdirAll(filepath.Dir(settingsPath()), 0755)
	data, _ := json.MarshalIndent(map[string]string{"downloadDir": dir}, "", "  ")
	_ = os.WriteFile(settingsPath(), data, 0644)
}
func loadDownloadDir() string {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return ""
	}
	var v map[string]string
	if json.Unmarshal(data, &v) != nil {
		return ""
	}
	return v["downloadDir"]
}

const (
	versionCacheFreshTTL  = 12 * time.Hour
	versionCacheMaxAgeTTL = 30 * 24 * time.Hour
)

func versionCachePath() string {
	return filepath.Join(roamingAppData(), "Wiz4rdFr0g", "version_cache.json")
}

func versionCacheKey(id, source string) string {
	return strings.ToLower(strings.TrimSpace(source) + "|" + strings.TrimSpace(id))
}

func loadVersionCache() {
	data, err := os.ReadFile(versionCachePath())
	if err != nil {
		return
	}
	var v map[string]versionCacheEntry
	if json.Unmarshal(data, &v) != nil {
		return
	}
	now := time.Now()
	versionCacheMu.Lock()
	defer versionCacheMu.Unlock()
	for k, entry := range v {
		if entry.ID == "" || entry.Source == "" || entry.UpdatedAt == 0 {
			continue
		}
		if now.Sub(time.Unix(entry.UpdatedAt, 0)) <= versionCacheMaxAgeTTL {
			versionCache[k] = entry
		}
	}
}

func getCachedVersions(id, source string) (versionCacheEntry, bool) {
	key := versionCacheKey(id, source)
	versionCacheMu.Lock()
	defer versionCacheMu.Unlock()
	entry, ok := versionCache[key]
	if !ok || entry.UpdatedAt == 0 {
		return versionCacheEntry{}, false
	}
	if time.Since(time.Unix(entry.UpdatedAt, 0)) > versionCacheMaxAgeTTL {
		delete(versionCache, key)
		return versionCacheEntry{}, false
	}
	entry.Versions = append([]string(nil), entry.Versions...)
	return entry, true
}

func cacheVersions(id, source string, versions []string) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(source) == "" {
		return
	}
	key := versionCacheKey(id, source)
	entry := versionCacheEntry{ID: id, Source: source, Versions: append([]string(nil), versions...), UpdatedAt: time.Now().Unix()}
	versionCacheMu.Lock()
	versionCache[key] = entry
	versionCacheMu.Unlock()
	go saveVersionCache()
}

func saveVersionCache() {
	versionCacheSaveMu.Lock()
	defer versionCacheSaveMu.Unlock()
	versionCacheMu.Lock()
	snapshot := make(map[string]versionCacheEntry, len(versionCache))
	for k, v := range versionCache {
		v.Versions = append([]string(nil), v.Versions...)
		snapshot[k] = v
	}
	versionCacheMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(versionCachePath()), 0755); err != nil {
		return
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return
	}
	tmp := versionCachePath() + ".tmp"
	if os.WriteFile(tmp, data, 0644) != nil {
		return
	}
	_ = os.Remove(versionCachePath())
	_ = os.Rename(tmp, versionCachePath())
}

func packageCachePath() string {
	return filepath.Join(roamingAppData(), "Wiz4rdFr0g", "package_cache.json")
}

func loadPackageCache() {
	data, err := os.ReadFile(packageCachePath())
	if err != nil {
		return
	}
	var v map[string]packageCacheEntry
	if json.Unmarshal(data, &v) != nil {
		return
	}
	now := time.Now().Unix()
	cacheMu.Lock()
	defer cacheMu.Unlock()
	for k, entry := range v {

		if entry.ID != "" && entry.Source != "" && entry.ResolverVersion == packageResolverVersion && (entry.UpdatedAt == 0 || now-entry.UpdatedAt <= int64((90*24*time.Hour)/time.Second)) {
			packageCache[k] = entry
		}
	}
}

func getCachedPackage(name string) (packageCacheEntry, bool) {
	key := normalizeSearch(name)
	cacheMu.Lock()
	defer cacheMu.Unlock()
	entry, ok := packageCache[key]
	if !ok || entry.ID == "" || entry.Source == "" || entry.ResolverVersion != packageResolverVersion {
		return packageCacheEntry{}, false
	}
	if entry.UpdatedAt != 0 && time.Now().Unix()-entry.UpdatedAt > int64((90*24*time.Hour)/time.Second) {
		delete(packageCache, key)
		return packageCacheEntry{}, false
	}
	return entry, true
}

func cachePackage(name, id, source string) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(source) == "" {
		return
	}
	key := normalizeSearch(name)
	cacheMu.Lock()
	packageCache[key] = packageCacheEntry{ID: id, Source: source, UpdatedAt: time.Now().Unix(), ResolverVersion: packageResolverVersion}
	_ = writePackageCacheLocked()
	cacheMu.Unlock()
}

func removeCachedPackage(name string) {
	key := normalizeSearch(name)
	cacheMu.Lock()
	delete(packageCache, key)
	_ = writePackageCacheLocked()
	cacheMu.Unlock()
}

func writePackageCacheLocked() error {
	if err := os.MkdirAll(filepath.Dir(packageCachePath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(packageCache, "", "  ")
	if err != nil {
		return err
	}
	tmp := packageCachePath() + ".tmp"
	if err = os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	_ = os.Remove(packageCachePath())
	return os.Rename(tmp, packageCachePath())
}
func metadataPath() string { return filepath.Join(roamingAppData(), "Wiz4rdFr0g", "installer.json") }
func roamingAppData() string {
	if v := os.Getenv("APPDATA"); v != "" {
		return v
	}
	d, _ := os.UserConfigDir()
	return d
}
func readInstallerMetadata() (installerMetadata, error) {
	var m installerMetadata
	data, err := os.ReadFile(metadataPath())
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if m.SourcePath == "" || m.SHA256 == "" {
		return m, fmt.Errorf("hiányos setup-metaadat")
	}
	return m, nil
}
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func browseFolder(current string) string {
	display := make([]uint16, 260)
	title := utf16Ptr("Mappa kiválasztása")
	bi := browseInfo{hwndOwner: mainWnd, pszDisplayName: &display[0], lpszTitle: title, ulFlags: BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	buf := make([]uint16, 32768)
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func addStatic(text string, x, y, w, h int, inst uintptr, group *[]uintptr) uintptr {
	hw := createWindow(0, "STATIC", text, WS_CHILD|WS_VISIBLE, x, y, w, h, mainWnd, 0, inst)
	setFont(hw)
	if group != nil {
		*group = append(*group, hw)
	}
	return hw
}
func addButton(text string, x, y, w, h int, id uintptr, inst uintptr, group *[]uintptr) uintptr {
	hw := createWindow(0, "BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, x, y, w, h, mainWnd, id, inst)
	setFont(hw)
	if group != nil {
		*group = append(*group, hw)
	}
	return hw
}
func createWindow(ex uint32, class, text string, style uint32, x, y, w, h int, parent, menu, inst uintptr) uintptr {
	r, _, _ := procCreateWindowExW.Call(uintptr(ex), uintptr(unsafe.Pointer(utf16Ptr(class))), uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, menu, inst, 0)
	return r
}
func setFont(h uintptr) {
	if h != 0 && defaultFont != 0 {
		procSendMessageW.Call(h, WM_SETFONT, defaultFont, 1)
	}
}
func setText(h uintptr, s string) {
	if h != 0 {
		procSetWindowTextW.Call(h, uintptr(unsafe.Pointer(utf16Ptr(s))))
	}
}
func getText(h uintptr) string {
	l, _, _ := procGetWindowTextLenW.Call(h)
	buf := make([]uint16, int(l)+1)
	procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func comboAdd(h uintptr, s string) {
	p := utf16Ptr(s)
	procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(p)))
}
func comboReset(h uintptr)         { procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0) }
func comboSelect(h uintptr, i int) { procSendMessageW.Call(h, CB_SETCURSEL, uintptr(i), 0) }
func comboSelectedText(h uintptr) string {
	idx, _, _ := procSendMessageW.Call(h, CB_GETCURSEL, 0, 0)
	if int32(idx) < 0 {
		return ""
	}
	l, _, _ := procSendMessageW.Call(h, CB_GETLBTEXTLEN, idx, 0)
	buf := make([]uint16, int(l)+1)
	procSendMessageW.Call(h, CB_GETLBTEXT, idx, uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}
func checkboxChecked(h uintptr) bool {
	r, _, _ := procSendMessageW.Call(h, BM_GETCHECK, 0, 0)
	return r == BST_CHECKED
}
func messageBox(h uintptr, text, title string, flags uintptr) int {
	r, _, _ := procMessageBoxW.Call(h, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), flags)
	return int(r)
}
func utf16Ptr(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func safeName(s string) string  { return regexp.MustCompile(`[\\/:*?"<>|]`).ReplaceAllString(s, "_") }
func safeFolderName(s string) string {
	s = strings.TrimSpace(safeName(s))
	s = strings.TrimRight(s, ". ")
	if s == "" {
		return "Program"
	}
	if len([]rune(s)) > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}
func versionFolder(v string) string {
	if v == "" {
		return "latest"
	}
	return safeName(v)
}
func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
func isSeparatorLine(s string) bool {
	if len(s) < 3 {
		return false
	}
	for _, r := range s {
		if r != '-' && r != '─' {
			return false
		}
	}
	return true
}
func shorten(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "..."
}
func quoteArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
func formatCommand(exe string, args []string) string {
	parts := []string{exe}
	for _, a := range args {
		parts = append(parts, quoteArg(a))
	}
	return strings.Join(parts, " ")
}
func boolCheck(v bool) uintptr {
	if v {
		return BST_CHECKED
	}
	return BST_UNCHECKED
}
func boolPtr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}
func firstN(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func init() {

	_ = sort.Strings
}
