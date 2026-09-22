package linuxpkg

import (
	"fmt"
	"net/url"
	"strings"
)

// ResolveAPTDownload handles the mirror+file URI emitted by hosted Ubuntu APT.
// The mirror list is read from APT configuration, never inferred from app names.
func ResolveAPTDownload(uri string, readFile func(string) ([]byte, error)) (string, error) {
	if strings.HasPrefix(uri, "https://") || strings.HasPrefix(uri, "http://") {
		return uri, nil
	}
	if !strings.HasPrefix(uri, "mirror+file:/etc/apt/") {
		return "", fmt.Errorf("unsupported APT download URI scheme")
	}
	prefix, suffix, ok := strings.Cut(strings.TrimPrefix(uri, "mirror+file:"), "/pool/")
	if !ok || strings.Contains(prefix, "..") || strings.Contains(suffix, "..") {
		return "", fmt.Errorf("invalid APT mirror file URI")
	}
	content, err := readFile(prefix)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		parsed, err := url.Parse(fields[0])
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			continue
		}
		return strings.TrimRight(fields[0], "/") + "/pool/" + suffix, nil
	}
	return "", fmt.Errorf("APT mirror list contains no HTTP source")
}
