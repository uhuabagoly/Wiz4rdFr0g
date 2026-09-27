package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
)

// NSIS firstheader from Source/exehead/fileform.h. Check the uninstall flag,
// signature and reserved flag bits before applying NSIS-specific arguments.
func nsisUninstallerHeader(data []byte) bool {
	if len(data) < 2 || string(data[:2]) != "MZ" {
		return false
	}
	magic := []byte{0xef, 0xbe, 0xad, 0xde, 'N', 'u', 'l', 'l', 's', 'o', 'f', 't', 'I', 'n', 's', 't'}
	for start := 4; start+len(magic)+8 <= len(data); {
		rel := bytes.Index(data[start:], magic)
		if rel < 0 {
			return false
		}
		at := start + rel
		if at+len(magic)+8 > len(data) {
			return false
		}
		flags := binary.LittleEndian.Uint32(data[at-4 : at])
		if flags&1 != 0 && flags & ^uint32(15) == 0 && binary.LittleEndian.Uint32(data[at+16:at+20]) > 0 {
			return true
		}
		start = at + len(magic)
	}
	return false
}

func nsisUninstallerFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 8<<20))
	return err == nil && nsisUninstallerHeader(data)
}
