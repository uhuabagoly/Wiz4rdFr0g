package main

import (
	"fmt"
	"regexp"
	"strings"
)

func splitWindowsArgs(s string) ([]string, error) {
	var args []string
	var b strings.Builder
	inQuotes := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuotes = !inQuotes
			continue
		}
		if (c == ' ' || c == '\t') && !inQuotes {
			if b.Len() > 0 {
				args = append(args, b.String())
				b.Reset()
			}
			continue
		}
		b.WriteByte(c)
	}
	if inQuotes {
		return nil, fmt.Errorf("lezáratlan idézőjel")
	}
	if b.Len() > 0 {
		args = append(args, b.String())
	}
	return args, nil
}

func executableCandidate(v string) bool {
	lower := strings.ToLower(strings.TrimSpace(v))
	for _, ext := range []string{".exe", ".com", ".bat", ".cmd"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func splitRegisteredCommandRaw(raw string) (string, []string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, fmt.Errorf("üres parancs")
	}
	parts, err := splitWindowsArgs(raw)
	if err == nil && len(parts) > 0 {
		// WinGet registers portable packages with its extensionless alias.
		if len(parts) == 4 && strings.EqualFold(parts[0], "winget") && parts[1] == "uninstall" && parts[2] == "--product-code" && regexp.MustCompile(`^[A-Za-z0-9_.-]+$`).MatchString(parts[3]) {
			return "winget.exe", append(parts[1:], "--silent", "--accept-source-agreements", "--disable-interactivity"), nil
		}
		exe := strings.Trim(strings.TrimSpace(parts[0]), `"`)
		if executableCandidate(exe) {
			return exe, parts[1:], nil
		}
	}
	lower := strings.ToLower(raw)
	bestEnd := -1
	for _, ext := range []string{".exe", ".com", ".bat", ".cmd"} {
		if i := strings.Index(lower, ext); i >= 0 {
			end := i + len(ext)
			if bestEnd < 0 || end < bestEnd {
				bestEnd = end
			}
		}
	}
	if bestEnd <= 0 {
		return "", nil, fmt.Errorf("nem található futtatható fájl a parancsban")
	}
	exe := strings.Trim(strings.TrimSpace(raw[:bestEnd]), `"`)
	rest := strings.TrimSpace(raw[bestEnd:])
	args := []string{}
	if rest != "" {
		args, err = splitWindowsArgs(rest)
		if err != nil {
			return "", nil, err
		}
	}
	return exe, args, nil
}

func extractMSIProductCode(v string) string {
	re := regexp.MustCompile(`\{[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}\}`)
	return re.FindString(v)
}

func registeredMSIProductCode(uninstall, quiet, key string, windowsInstaller bool) string {
	for _, command := range []string{uninstall, quiet} {
		exe, _, err := splitRegisteredCommandRaw(command)
		base := strings.ToLower(exe)
		if at := strings.LastIndexAny(base, `/\`); at >= 0 {
			base = base[at+1:]
		}
		if err == nil && base == "msiexec.exe" {
			return extractMSIProductCode(command)
		}
	}
	if windowsInstaller {
		return extractMSIProductCode(key)
	}
	return ""
}
