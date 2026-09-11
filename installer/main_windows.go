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
	"strings"
	"syscall"
	"unsafe"
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

var (
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procShellExecuteExW     = shell32.NewProc("ShellExecuteExW")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess  = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
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
	elevated := false
	for _, a := range os.Args[1:] {
		if a == "--elevated" {
			elevated = true
		}
	}
	if !elevated {
		code, err := elevateSelf()
		if err != nil {
			messageBox(appName+" telepítő", "A telepítéshez rendszergazdai jogosultság szükséges.\n\n"+err.Error(), 0x10)
			return
		}
		if code != 0 {
			messageBox(appName+" telepítő", fmt.Sprintf("A telepítés nem fejeződött be sikeresen. Kilépési kód: %d", code), 0x10)
			return
		}
		if err = launchInstalledApp(); err != nil {
			messageBox(appName, "A telepítés elkészült, de az alkalmazás automatikus indítása nem sikerült.\n\n"+err.Error()+"\n\nA Start menüből elindítható.", 0x30)
		}
		return
	}
	if err := install(); err != nil {
		messageBox(appName+" telepítő", "A telepítés sikertelen:\n\n"+err.Error(), 0x10)
		os.Exit(1)
	}
	os.Exit(0)
}

func install() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, _ = filepath.Abs(self)
	hash, err := fileSHA256(self)
	if err != nil {
		return fmt.Errorf("telepítő ellenőrzése: %w", err)
	}
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	installDir := filepath.Join(pf, "Wiz4rd Fr0g")
	target := filepath.Join(installDir, "Wiz4rdFr0g.exe")
	_ = migrateLegacyInstall(pf)
	if err = os.MkdirAll(installDir, 0755); err != nil {
		return err
	}
	_ = hiddenCmd("taskkill.exe", "/IM", "Wiz4rdFr0g.exe", "/F").Run()
	data, err := payloadFS.ReadFile("payload/Wiz4rdFr0g.exe")
	if err != nil {
		return err
	}
	tmp := target + ".new"
	if err = os.WriteFile(tmp, data, 0755); err != nil {
		return err
	}
	_ = os.Remove(target)
	if err = os.Rename(tmp, target); err != nil {
		return err
	}

	cfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cfgDir := filepath.Join(cfg, "Wiz4rdFr0g")
	if err = os.MkdirAll(cfgDir, 0755); err != nil {
		return err
	}
	meta, _ := json.MarshalIndent(installerMeta{SourcePath: self, SHA256: hash, Publisher: publisher, Version: appVersion}, "", "  ")
	if err = os.WriteFile(filepath.Join(cfgDir, "installer.json"), meta, 0644); err != nil {
		return err
	}
	_ = createShortcut(target, installDir)
	_ = registerInstalledApp(target, installDir)
	return nil
}

func elevateSelf() (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return -1, err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString("--elevated")
	info := shellExecuteInfo{fMask: 0x00000040, lpVerb: verb, lpFile: file, lpParameters: params, nShow: 1}
	info.cbSize = uint32(unsafe.Sizeof(info))
	r1, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return -1, fmt.Errorf("rendszergazdai indítás sikertelen: %v", callErr)
	}
	if info.hProcess == 0 {
		return -1, fmt.Errorf("az emelt telepítő nem adott process handle-t")
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

func launchInstalledApp() error {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	target := filepath.Join(pf, "Wiz4rd Fr0g", "Wiz4rdFr0g.exe")
	if _, err := os.Stat(target); err != nil {
		return err
	}
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	return cmd.Start()
}
func createShortcut(target, workDir string) error {
	ps := `$w=New-Object -ComObject WScript.Shell; $p=Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Wiz4rd Fr0g.lnk'; $s=$w.CreateShortcut($p); $s.TargetPath='` + strings.ReplaceAll(target, "'", "''") + `'; $s.WorkingDirectory='` + strings.ReplaceAll(workDir, "'", "''") + `'; $s.IconLocation='` + strings.ReplaceAll(target, "'", "''") + `,0'; $s.Description='Wiz4rd Fr0g - ` + publisher + `'; $s.Save()`
	return hiddenCmd("powershell.exe", "-NoProfile", "-Command", ps).Run()
}
func registerInstalledApp(target, installDir string) error {
	key := `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Wiz4rdFr0g`
	vals := [][]string{{"ADD", key, "/v", "DisplayName", "/t", "REG_SZ", "/d", appName, "/f"}, {"ADD", key, "/v", "DisplayVersion", "/t", "REG_SZ", "/d", appVersion, "/f"}, {"ADD", key, "/v", "Publisher", "/t", "REG_SZ", "/d", publisher, "/f"}, {"ADD", key, "/v", "URLInfoAbout", "/t", "REG_SZ", "/d", publisher, "/f"}, {"ADD", key, "/v", "InstallLocation", "/t", "REG_SZ", "/d", installDir, "/f"}, {"ADD", key, "/v", "DisplayIcon", "/t", "REG_SZ", "/d", target, "/f"}, {"ADD", key, "/v", "NoModify", "/t", "REG_DWORD", "/d", "1", "/f"}, {"ADD", key, "/v", "NoRepair", "/t", "REG_DWORD", "/d", "1", "/f"}}
	for _, a := range vals {
		if err := hiddenCmd("reg.exe", a...).Run(); err != nil {
			return err
		}
	}
	return nil
}
func migrateLegacyInstall(programFiles string) error {

	_ = hiddenCmd("taskkill.exe", "/IM", "LetoltoKozpont.exe", "/F").Run()
	oldShortcut := filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs\LetöltőKözpont.lnk`)
	_ = os.Remove(oldShortcut)
	_ = hiddenCmd("reg.exe", "DELETE", `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\LetoltoKozpont`, "/f").Run()

	cfg, err := os.UserConfigDir()
	if err == nil {
		oldCfg := filepath.Join(cfg, "LetoltoKozpont")
		newCfg := filepath.Join(cfg, "Wiz4rdFr0g")
		if _, statErr := os.Stat(oldCfg); statErr == nil {
			_ = os.MkdirAll(newCfg, 0755)
			_ = copyIfMissing(filepath.Join(oldCfg, "settings.json"), filepath.Join(newCfg, "settings.json"))
		}
	}
	oldDir := filepath.Join(programFiles, "LetoltoKozpont")
	_ = os.RemoveAll(oldDir)
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
func messageBox(title, text string, flags uintptr) {
	u := syscall.NewLazyDLL("user32.dll")
	p := u.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	tt, _ := syscall.UTF16PtrFromString(title)
	p.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(tt)), flags)
}
