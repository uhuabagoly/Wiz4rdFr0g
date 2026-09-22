//go:build linux

package main

import "testing"

func TestOllamaArchiveOnlyContainsOwnedInstallationPaths(t *testing.T) {
	if err := validateOllamaPaths("bin/\nbin/ollama\nlib/\nlib/ollama/\nlib/ollama/libggml.so\n"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"bin/ollama\n../etc/passwd", "bin/ollama\n/etc/passwd", "bin/ollama\nlib/other-app/file", "bin/ollama\nlib/ollama/../../etc/passwd", "lib/ollama/file"} {
		if err := validateOllamaPaths(bad); err == nil {
			t.Fatalf("accepted unowned/incomplete payload %q", bad)
		}
	}
}
