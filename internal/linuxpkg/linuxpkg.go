package linuxpkg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Spec struct {
	Name      string
	Args      []string
	NeedsRoot bool
}

var (
	packageIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+_:@/=-]*$`)
	versionPattern   = regexp.MustCompile(`^[A-Za-z0-9.+:~_\-]+$`)
	remotePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

func Validate(provider, packageID, remote, version string) error {
	if !packageIDPattern.MatchString(packageID) {
		return fmt.Errorf("invalid exact package identifier %q", packageID)
	}
	if version != "" && version != "Legújabb" && !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid version %q", version)
	}
	if provider == "flatpak" && remote != "" && !remotePattern.MatchString(remote) {
		return fmt.Errorf("invalid Flatpak remote %q", remote)
	}
	switch provider {
	case "apt-get", "dnf", "pacman", "zypper", "flatpak", "uv-python":
		return nil
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}
}

func Install(provider, packageID, remote, version string) (Spec, error) {
	if err := Validate(provider, packageID, remote, version); err != nil {
		return Spec{}, err
	}
	switch provider {
	case "uv-python":
		return Spec{Name: "uv", Args: []string{"--no-config", "python", "install", "--no-bin", packageID}}, nil
	case "apt-get":
		pkg := packageID
		if version != "" && version != "Legújabb" {
			pkg += "=" + version
		}
		return Spec{Name: "apt-get", Args: []string{"install", "-y", pkg}, NeedsRoot: true}, nil
	case "dnf":
		pkg := packageID
		if version != "" && version != "Legújabb" {
			pkg += "-" + version
		}
		return Spec{Name: "dnf", Args: []string{"install", "-y", pkg}, NeedsRoot: true}, nil
	case "pacman":
		return Spec{Name: "pacman", Args: []string{"-S", "--noconfirm", packageID}, NeedsRoot: true}, nil
	case "zypper":
		pkg := packageID
		if version != "" && version != "Legújabb" {
			pkg += "=" + version
		}
		return Spec{Name: "zypper", Args: []string{"--non-interactive", "install", pkg}, NeedsRoot: true}, nil
	case "flatpak":
		if remote == "" {
			return Spec{}, fmt.Errorf("Flatpak remote missing for %q", packageID)
		}
		return Spec{Name: "flatpak", Args: []string{"install", "--user", "-y", remote, packageID}}, nil
	}
	panic("unreachable")
}

func Remove(provider, packageID, remote string) (Spec, error) {
	if err := Validate(provider, packageID, remote, ""); err != nil {
		return Spec{}, err
	}
	switch provider {
	case "uv-python":
		return Spec{Name: "uv", Args: []string{"--no-config", "python", "uninstall", packageID}}, nil
	case "apt-get":
		return Spec{Name: "apt-get", Args: []string{"remove", "-y", packageID}, NeedsRoot: true}, nil
	case "dnf":
		return Spec{Name: "dnf", Args: []string{"remove", "-y", packageID}, NeedsRoot: true}, nil
	case "pacman":
		return Spec{Name: "pacman", Args: []string{"-R", "--noconfirm", packageID}, NeedsRoot: true}, nil
	case "zypper":
		return Spec{Name: "zypper", Args: []string{"--non-interactive", "remove", packageID}, NeedsRoot: true}, nil
	case "flatpak":
		return Spec{Name: "flatpak", Args: []string{"uninstall", "--user", "-y", packageID}}, nil
	}
	panic("unreachable")
}

func Detect(provider, packageID, remote string) (Spec, error) {
	if err := Validate(provider, packageID, remote, ""); err != nil {
		return Spec{}, err
	}
	switch provider {
	case "uv-python":
		return Spec{Name: "uv", Args: []string{"--no-config", "python", "find", "--managed-python", "--no-python-downloads", packageID}}, nil
	case "apt-get":
		return Spec{Name: "dpkg-query", Args: []string{"-W", "-f=${Status}", packageID}}, nil
	case "dnf", "zypper":
		return Spec{Name: "rpm", Args: []string{"-q", "--", packageID}}, nil
	case "pacman":
		return Spec{Name: "pacman", Args: []string{"-Q", "--", packageID}}, nil
	case "flatpak":
		return Spec{Name: "flatpak", Args: []string{"info", "--user", packageID}}, nil
	}
	panic("unreachable")
}

func DetectionSuccess(provider string, output []byte, exitOK bool) bool {
	if !exitOK {
		return false
	}
	if provider == "apt-get" {
		return strings.TrimSpace(string(output)) == "install ok installed"
	}
	return true
}

// Inventory queries must succeed before absence can be established. An exact
// lookup's nonzero status can also mean a broken database or denied access.
func Inventory(provider string) (Spec, error) {
	switch provider {
	case "uv-python":
		return Spec{Name: "uv", Args: []string{"--no-config", "python", "list", "--only-installed", "--managed-python", "--output-format", "json"}}, nil
	case "apt-get":
		return Spec{Name: "dpkg-query", Args: []string{"-W", "-f=${Package}\t${Status}\n"}}, nil
	case "dnf", "zypper":
		return Spec{Name: "rpm", Args: []string{"-qa", "--qf", "%{NAME}\n"}}, nil
	case "pacman":
		return Spec{Name: "pacman", Args: []string{"-Qq"}}, nil
	case "flatpak":
		return Spec{Name: "flatpak", Args: []string{"list", "--user", "--app", "--columns=application"}}, nil
	default:
		return Spec{}, fmt.Errorf("unsupported provider %q", provider)
	}
}

func InventoryContains(provider, packageID string, output []byte, exitCode int) (bool, error) {
	if _, err := Inventory(provider); err != nil {
		return false, err
	}
	if exitCode != 0 {
		return false, fmt.Errorf("%s inventory failed (exit %d); installed state is unknown", provider, exitCode)
	}
	if provider == "uv-python" {
		var rows []struct {
			Key  string `json:"key"`
			Path string `json:"path"`
		}
		if json.Unmarshal(output, &rows) != nil || !strings.HasPrefix(strings.TrimSpace(string(output)), "[") {
			return false, fmt.Errorf("malformed managed Python inventory")
		}
		for _, row := range rows {
			if row.Key == packageID && row.Path != "" {
				return true, nil
			}
		}
		return false, nil
	}
	found := false
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if provider == "apt-get" {
			parts := strings.Split(line, "\t")
			if len(parts) != 2 || len(strings.Fields(parts[1])) != 3 {
				return false, fmt.Errorf("malformed dpkg inventory; installed state is unknown")
			}
			if parts[0] == packageID {
				status := strings.Fields(parts[1])[2]
				switch status {
				case "installed":
					if strings.Fields(parts[1])[1] != "ok" {
						return false, fmt.Errorf("package %s requires repair", packageID)
					}
					found = true
				case "not-installed", "config-files":
				default:
					return false, fmt.Errorf("package %s is in transitional state %s", packageID, status)
				}
			}
		} else {
			if len(strings.Fields(line)) != 1 {
				return false, fmt.Errorf("malformed %s inventory", provider)
			}
			if line == packageID {
				found = true
			}
		}
	}
	return found, nil
}
