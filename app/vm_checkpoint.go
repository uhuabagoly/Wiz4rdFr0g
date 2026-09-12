package main

import (
	"os"
	"path/filepath"
)

func vmAtomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return err }
	f, err := os.CreateTemp(filepath.Dir(path), ".checkpoint-*")
	if err != nil { return err }
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(data); err != nil { f.Close(); return err }
	if err := f.Sync(); err != nil { f.Close(); return err }
	if err := f.Close(); err != nil { return err }
	return os.Rename(name, path)
}
