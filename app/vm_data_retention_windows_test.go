//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataRetentionRejectsChangedOrDeletedWitness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "witness.txt")
	if err := os.WriteFile(path, []byte("retained"), 0600); err != nil {
		t.Fatal(err)
	}
	n, digest, err := vmFileDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	p := vmFilesystemProof{PreservedData: []vmPreservedDataProof{{Path: path, Bytes: n, SHA256: digest}}}
	if err := vmVerifyDataRetention(&p); err != nil || !p.PreservedData[0].Preserved {
		t.Fatal("unchanged witness rejected", err)
	}
	if err := os.WriteFile(path, []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	if vmVerifyDataRetention(&p) == nil || p.PreservedData[0].Preserved {
		t.Fatal("same-length alteration accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if vmVerifyDataRetention(&p) == nil || p.PreservedData[0].Preserved {
		t.Fatal("deleted data accepted")
	}
}
