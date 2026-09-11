//go:build windows

package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"wiz4rdfr0g.local/fullcatalog/internal/selfinstall"
)

//go:embed payload/Wiz4rdFr0g.exe
var payloadFS embed.FS

const appName = "Wiz4rd Fr0g"
const appVersion = "0.6.23"
const publisher = "https://github.com/uhuabagoly"

type installerMeta struct {
	SourcePath string `json:"sourcePath"`
	SHA256     string `json:"sha256"`
	Publisher  string `json:"publisher"`
	Version    string `json:"version"`
}

type cliOptions struct {
	Elevated   bool
	Uninstall  bool
	Quiet      bool
	Cleanup    bool
	CleanupDir string
}

var (
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procShellExecuteExW     = shell32.NewProc("ShellExecuteExW")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess  = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
	procMoveFileExW         = kernel32.NewProc("MoveFileExW")
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

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		messageBox(appName, err.Error(), 0x10)
		os.Exit(2)
	}
	if opts.Cleanup {
		if err := cleanupInstallDir(opts.CleanupDir, opts.Quiet); err != nil {
			os.Exit(1)
		}
		return
	}
	if opts.Uninstall {
		os.Exit(runUninstall(opts))
	}
	os.Exit(runInstall(opts))
}

func runInstall(opts cliOptions) int {
	if !opts.Elevated {
		args := []string{"--elevated"}
		if opts.Quiet {
			args = append(args, "--quiet")
		}
		code, err := elevateSelf(args...)
		if err != nil {
			if !opts.Quiet {
				messageBox(appName+" telepítő", "A telepítéshez rendszergazdai jogosultság szükséges.\n\n"+err.Error(), 0x10)
			}
			return 1
		}
		if code != 0 {
			if !opts.Quiet {
				messageBox(appName+" telepítő", fmt.Sprintf("A telepítés nem fejeződött be sikeresen. Kilépési kód: %d", code), 0x10)
			}
			return code
		}
		if err := finalizeOriginalUserInstall(); err != nil {
			if !opts.Quiet {
				messageBox(appName+" telepítő", "A gépszintű telepítés elkészült, de a felhasználói profil beállítása sikertelen:\n\n"+err.Error(), 0x10)
			}
			return 2
		}
		if !opts.Quiet {
			if err := launchInstalledApp(); err != nil {
				messageBox(appName, "A telepítés elkészült, de az alkalmazás automatikus indítása nem sikerült.\n\n"+err.Error()+"\n\nA Start menüből elindítható.", 0x30)
			}
		}
		return 0
	}
	if err := installMachineScope(); err != nil {
		if !opts.Quiet {
			messageBox(appName+" telepítő", "A telepítés sikertelen:\n\n"+err.Error(), 0x10)
		}
		return 1
	}
	return 0
}

func runUninstall(opts cliOptions) int {
	if !opts.Elevated {
		if !opts.Quiet {
			const idYes = 6
			if messageBox(appName+" eltávolító", "Biztosan eltávolítod a Wiz4rd Fr0g alkalmazást?\n\nA személyes beállítások megmaradnak.", 0x24) != idYes {
				return 0
			}
		}
		args := []string{"--uninstall", "--elevated"}
		if opts.Quiet {
			args = append(args, "--quiet")
		}
		code, err := elevateSelf(args...)
		if err != nil {
			if !opts.Quiet {
				messageBox(appName+" eltávolító", "Az eltávolításhoz rendszergazdai jogosultság szükséges.\n\n"+err.Error(), 0x10)
			}
			return 1
		}
		if code != 0 {
			if !opts.Quiet {
				messageBox(appName+" eltávolító", fmt.Sprintf("Az eltávolítás nem fejeződött be sikeresen. Kilépési kód: %d", code), 0x10)
			}
			return code
		}
		if err := removeOriginalUserArtifacts(); err != nil {
			if !opts.Quiet {
				messageBox(appName+" eltávolító", "A program eltávolítása elkészült, de néhány felhasználói telepítési metaadat nem törölhető:\n\n"+err.Error(), 0x30)
			}
			return 2
		}
		if !opts.Quiet {
			messageBox(appName+" eltávolító", "A Wiz4rd Fr0g eltávolítása elkészült. A személyes beállítások megmaradtak.", 0x40)
		}
		return 0
	}
	if err := uninstallMachineScope(opts.Quiet); err != nil {
		return 1
	}
	return 0
}

func installMachineScope() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, _ = filepath.Abs(self)
	pf := programFilesDir()
	installDir := filepath.Join(pf, selfinstall.AppDirName)
	target := filepath.Join(installDir, selfinstall.AppExeName)
	uninstaller := filepath.Join(installDir, selfinstall.UninstallerName)

	if err := migrateLegacyMachine(pf); err != nil {
		return fmt.Errorf("legacy install cleanup: %w", err)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return err
	}
	if err := stopExactExecutable(target); err != nil {
		return fmt.Errorf("running application: %w", err)
	}
	data, err := payloadFS.ReadFile("payload/Wiz4rdFr0g.exe")
	if err != nil {
		return err
	}
	if err := replaceFile(target, data, 0755); err != nil {
		return fmt.Errorf("application payload: %w", err)
	}
	if err := copyFileAtomic(self, uninstaller, 0755); err != nil {
		return fmt.Errorf("uninstaller: %w", err)
	}
	if err := registerInstalledApp(target, installDir, uninstaller); err != nil {
		return fmt.Errorf("Windows uninstall registration: %w", err)
	}
	return nil
}

func finalizeOriginalUserInstall() error {
	if err := migrateLegacyUser(); err != nil {
		return err
	}
	pf := programFilesDir()
	installDir := filepath.Join(pf, selfinstall.AppDirName)
	target := filepath.Join(installDir, selfinstall.AppExeName)
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("installed application missing: %w", err)
	}
	if err := createShortcut(target, installDir); err != nil {
		return fmt.Errorf("Start menu shortcut: %w", err)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, _ = filepath.Abs(self)
	hash, err := fileSHA256(self)
	if err != nil {
		return fmt.Errorf("installer metadata hash: %w", err)
	}
	return writeInstallerMetadata(self, hash)
}

func uninstallMachineScope(quiet bool) error {
	pf := programFilesDir()
	installDir := filepath.Join(pf, selfinstall.AppDirName)
	target := filepath.Join(installDir, selfinstall.AppExeName)

	if err := stopExactExecutable(target); err != nil {
		return fmt.Errorf("running application: %w", err)
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove application: %w", err)
	}
	if err := hiddenCmd("reg.exe", "DELETE", selfinstall.RegistryKey, "/f").Run(); err != nil {
		// reg.exe returns non-zero when the key is already absent. Verify before failing.
		if registryKeyExists(selfinstall.RegistryKey) {
			return fmt.Errorf("remove uninstall registry entry: %w", err)
		}
	}
	if err := startCleanupHelper(installDir, quiet); err != nil {
		return fmt.Errorf("deferred install-directory cleanup: %w", err)
	}
	return nil
}

func startCleanupHelper(installDir string, quiet bool) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "Wiz4rdFr0g-uninstall-")
	if err != nil {
		return err
	}
	helper := filepath.Join(tmpDir, "cleanup.exe")
	if err := copyFileAtomic(self, helper, 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	args := []string{"--cleanup", "--dir", installDir}
	if quiet {
		args = append(args, "--quiet")
	}
	cmd := exec.Command(helper, args...)
	cmd.Dir = tmpDir
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return nil
}

func cleanupInstallDir(requestedDir string, quiet bool) error {
	expected := filepath.Join(programFilesDir(), selfinstall.AppDirName)
	if !selfinstall.SameWindowsPath(requestedDir, expected) {
		return fmt.Errorf("refusing cleanup outside expected install directory: %s", requestedDir)
	}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = os.RemoveAll(expected)
		if lastErr == nil {
			if _, err := os.Stat(expected); os.IsNotExist(err) {
				scheduleOwnCleanup()
				return nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	// The running original uninstaller or an external scanner can temporarily
	// keep Uninstall.exe locked. Fall back to Windows' reboot deletion mechanism.
	uninstaller := filepath.Join(expected, selfinstall.UninstallerName)
	_ = scheduleDeleteOnReboot(uninstaller)
	_ = scheduleDeleteOnReboot(expected)
	marker := filepath.Join(os.TempDir(), "Wiz4rdFr0g_reboot_required.txt")
	_ = os.WriteFile(marker, []byte("Wiz4rd Fr0g uninstall cleanup requires a Windows restart.\r\n"), 0644)
	if !quiet {
		messageBox(appName+" eltávolító", "A fő alkalmazás eltávolítása elkészült, de egy zárolt eltávolítófájl miatt a végső takarításhoz Windows-újraindítás szükséges.", 0x30)
	}
	scheduleOwnCleanup()
	if lastErr != nil {
		return lastErr
	}
	return nil
}

func scheduleOwnCleanup() {
	self, err := os.Executable()
	if err != nil {
		return
	}
	_ = scheduleDeleteOnReboot(self)
	_ = scheduleDeleteOnReboot(filepath.Dir(self))
}

func scheduleDeleteOnReboot(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	const movefileDelayUntilReboot = 0x00000004
	r, _, callErr := procMoveFileExW.Call(uintptr(unsafe.Pointer(p)), 0, movefileDelayUntilReboot)
	if r == 0 {
		return callErr
	}
	return nil
}

func parseOptions(args []string) (cliOptions, error) {
	var o cliOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--elevated":
			o.Elevated = true
		case "--uninstall":
			o.Uninstall = true
		case "--quiet":
			o.Quiet = true
		case "--cleanup":
			o.Cleanup = true
		case "--dir":
			if i+1 >= len(args) {
				return o, fmt.Errorf("--dir requires a value")
			}
			i++
			o.CleanupDir = args[i]
		default:
			return o, fmt.Errorf("unknown argument: %s", args[i])
		}
	}
	if o.Cleanup && strings.TrimSpace(o.CleanupDir) == "" {
		return o, fmt.Errorf("cleanup directory is required")
	}
	return o, nil
}

func elevateSelf(args ...string) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return -1, err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString(joinWindowsArgs(args))
	info := shellExecuteInfo{fMask: 0x00000040, lpVerb: verb, lpFile: file, lpParameters: params, nShow: 1}
	info.cbSize = uint32(unsafe.Sizeof(info))
	r1, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return -1, fmt.Errorf("rendszergazdai indítás sikertelen: %v", callErr)
	}
	if info.hProcess == 0 {
		return -1, fmt.Errorf("az emelt folyamat nem adott process handle-t")
	}
	defer procCloseHandle.Call(info.hProcess)
	procWaitForSingleObject.Call(info.hProcess, 0xFFFFFFFF)
	var code uint32
	r, _, exitErr := procGetExitCodeProcess.Call(info.hProcess, uintptr(unsafe.Pointer(&code)))
	if r == 0 {
		return -1, fmt.Errorf("kilépési kód lekérése sikertelen: %v", exitErr)
	}
	return int(code), nil
}

func joinWindowsArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, a := range args {
		if strings.ContainsAny(a, " \t\"") {
			quoted = append(quoted, strconv.Quote(a))
		} else {
			quoted = append(quoted, a)
		}
	}
	return strings.Join(quoted, " ")
}

func launchInstalledApp() error {
	target := filepath.Join(programFilesDir(), selfinstall.AppDirName, selfinstall.AppExeName)
	if _, err := os.Stat(target); err != nil {
		return err
	}
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	return cmd.Start()
}

func createShortcut(target, workDir string) error {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return fmt.Errorf("APPDATA is unavailable")
	}
	shortcut := filepath.Join(appData, selfinstall.StartMenuLink)
	if err := os.MkdirAll(filepath.Dir(shortcut), 0755); err != nil {
		return err
	}
	ps := `$w=New-Object -ComObject WScript.Shell; $p='` + escapePowerShellSingle(shortcut) + `'; $s=$w.CreateShortcut($p); $s.TargetPath='` + escapePowerShellSingle(target) + `'; $s.WorkingDirectory='` + escapePowerShellSingle(workDir) + `'; $s.IconLocation='` + escapePowerShellSingle(target) + `,0'; $s.Description='Wiz4rd Fr0g - ` + escapePowerShellSingle(publisher) + `'; $s.Save()`
	return hiddenCmd("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}

func registerInstalledApp(target, installDir, uninstaller string) error {
	key := selfinstall.RegistryKey
	interactive := selfinstall.UninstallCommand(uninstaller, false)
	quiet := selfinstall.UninstallCommand(uninstaller, true)
	estimatedKB := installedSizeKB(target, uninstaller)
	vals := [][]string{
		{"ADD", key, "/v", "DisplayName", "/t", "REG_SZ", "/d", appName, "/f"},
		{"ADD", key, "/v", "DisplayVersion", "/t", "REG_SZ", "/d", appVersion, "/f"},
		{"ADD", key, "/v", "Publisher", "/t", "REG_SZ", "/d", publisher, "/f"},
		{"ADD", key, "/v", "URLInfoAbout", "/t", "REG_SZ", "/d", publisher, "/f"},
		{"ADD", key, "/v", "InstallLocation", "/t", "REG_SZ", "/d", installDir, "/f"},
		{"ADD", key, "/v", "DisplayIcon", "/t", "REG_SZ", "/d", target, "/f"},
		{"ADD", key, "/v", "UninstallString", "/t", "REG_SZ", "/d", interactive, "/f"},
		{"ADD", key, "/v", "QuietUninstallString", "/t", "REG_SZ", "/d", quiet, "/f"},
		{"ADD", key, "/v", "InstallDate", "/t", "REG_SZ", "/d", time.Now().Format("20060102"), "/f"},
		{"ADD", key, "/v", "EstimatedSize", "/t", "REG_DWORD", "/d", strconv.FormatInt(estimatedKB, 10), "/f"},
		{"ADD", key, "/v", "NoModify", "/t", "REG_DWORD", "/d", "1", "/f"},
		{"ADD", key, "/v", "NoRepair", "/t", "REG_DWORD", "/d", "1", "/f"},
	}
	for _, a := range vals {
		if err := hiddenCmd("reg.exe", a...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func registryKeyExists(key string) bool {
	return hiddenCmd("reg.exe", "QUERY", key).Run() == nil
}

func installedSizeKB(paths ...string) int64 {
	var total int64
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			total += st.Size()
		}
	}
	if total == 0 {
		return 1
	}
	return (total + 1023) / 1024
}

func writeInstallerMetadata(sourcePath, hash string) error {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cfgDir := filepath.Join(cfg, "Wiz4rdFr0g")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return err
	}
	meta, err := json.MarshalIndent(installerMeta{SourcePath: sourcePath, SHA256: hash, Publisher: publisher, Version: appVersion}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfgDir, "installer.json"), meta, 0644)
}

func removeOriginalUserArtifacts() error {
	var errs []string
	if appData := os.Getenv("APPDATA"); appData != "" {
		shortcut := filepath.Join(appData, selfinstall.StartMenuLink)
		if err := os.Remove(shortcut); err != nil && !os.IsNotExist(err) {
			errs = append(errs, "shortcut: "+err.Error())
		}
	}
	if cfg, err := os.UserConfigDir(); err == nil {
		cfgDir := filepath.Join(cfg, "Wiz4rdFr0g")
		meta := filepath.Join(cfgDir, "installer.json")
		if err := os.Remove(meta); err != nil && !os.IsNotExist(err) {
			errs = append(errs, "installer metadata: "+err.Error())
		}
		_ = os.Remove(cfgDir) // succeeds only if no personal settings remain
	}
	if len(errs) != 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func migrateLegacyMachine(programFiles string) error {
	oldDir := filepath.Join(programFiles, selfinstall.LegacyDirName)
	oldExe := filepath.Join(oldDir, selfinstall.LegacyExeName)
	if err := stopExactExecutable(oldExe); err != nil {
		return err
	}
	if err := hiddenCmd("reg.exe", "DELETE", selfinstall.LegacyRegistryKey, "/f").Run(); err != nil && registryKeyExists(selfinstall.LegacyRegistryKey) {
		return err
	}
	if err := os.RemoveAll(oldDir); err != nil {
		return err
	}
	return nil
}

func migrateLegacyUser() error {
	if appData := os.Getenv("APPDATA"); appData != "" {
		oldShortcut := filepath.Join(appData, `Microsoft\Windows\Start Menu\Programs\LetöltőKözpont.lnk`)
		_ = os.Remove(oldShortcut)
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	oldCfg := filepath.Join(cfg, "LetoltoKozpont")
	newCfg := filepath.Join(cfg, "Wiz4rdFr0g")
	if _, statErr := os.Stat(oldCfg); statErr == nil {
		if err := os.MkdirAll(newCfg, 0755); err != nil {
			return err
		}
		if err := copyIfMissing(filepath.Join(oldCfg, "settings.json"), filepath.Join(newCfg, "settings.json")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func copyIfMissing(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0644)
}

func replaceFile(target string, data []byte, mode os.FileMode) error {
	tmp := target + ".new"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func copyFileAtomic(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func programFilesDir() string {
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		return pf
	}
	return `C:\Program Files`
}

func escapePowerShellSingle(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

func hiddenCmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return c
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

func messageBox(title, text string, flags uintptr) uintptr {
	u := syscall.NewLazyDLL("user32.dll")
	p := u.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	tt, _ := syscall.UTF16PtrFromString(title)
	r, _, _ := p.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(tt)), flags)
	return r
}
