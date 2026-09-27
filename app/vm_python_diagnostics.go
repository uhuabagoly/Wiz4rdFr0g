package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func vmPythonInstallerLogs(started, explicit string) map[string]string {
	start, err := time.Parse(time.RFC3339Nano, started)
	if err != nil {
		return nil
	}
	paths, _ := filepath.Glob(filepath.Join(os.TempDir(), "Python*.log"))
	paths = append(paths, explicit)
	sort.Strings(paths)
	out := map[string]string{}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.ModTime().Before(start) || len(out) >= 8 {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		if info.Size() > 16000 {
			_, _ = f.Seek(info.Size()-16000, io.SeekStart)
		}
		b, err := io.ReadAll(io.LimitReader(f, 16000))
		f.Close()
		if err == nil {
			out[path] = string(b)
		}
	}
	return out
}
