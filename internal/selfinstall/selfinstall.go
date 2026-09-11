package selfinstall

import (
	"path"
	"strings"
)

const (
	AppDirName        = "Wiz4rd Fr0g"
	AppExeName        = "Wiz4rdFr0g.exe"
	UninstallerName   = "Uninstall.exe"
	RegistryKey       = `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Wiz4rdFr0g`
	StartMenuLink     = `Microsoft\Windows\Start Menu\Programs\Wiz4rd Fr0g.lnk`
	LegacyDirName     = "LetoltoKozpont"
	LegacyExeName     = "LetoltoKozpont.exe"
	LegacyRegistryKey = `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\LetoltoKozpont`
)

// NormalizeWindowsPath returns a case-insensitive, slash-normalized representation
// suitable for comparing absolute Windows file paths on any host OS.
func NormalizeWindowsPath(v string) string {
	v = strings.TrimSpace(strings.Trim(v, `"`))
	v = strings.ReplaceAll(v, `\`, "/")
	v = path.Clean(v)
	if v == "." {
		return ""
	}
	return strings.ToLower(v)
}

func SameWindowsPath(a, b string) bool {
	na, nb := NormalizeWindowsPath(a), NormalizeWindowsPath(b)
	return na != "" && na == nb
}

func PathWithinWindowsRoot(candidate, root string) bool {
	nc, nr := NormalizeWindowsPath(candidate), NormalizeWindowsPath(root)
	if nc == "" || nr == "" {
		return false
	}
	if nc == nr {
		return true
	}
	return strings.HasPrefix(nc, strings.TrimRight(nr, "/")+"/")
}

func QuoteWindowsArg(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
}

func UninstallCommand(uninstaller string, quiet bool) string {
	cmd := QuoteWindowsArg(uninstaller) + " --uninstall"
	if quiet {
		cmd += " --quiet"
	}
	return cmd
}
