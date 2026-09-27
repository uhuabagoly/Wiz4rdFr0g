package main

import (
	"encoding/binary"
	"testing"
)

func TestNSISRequiresUninstallerHeader(t *testing.T) {
	data := make([]byte, 128)
	copy(data, "MZ")
	copy(data[68:], []byte{0xef, 0xbe, 0xad, 0xde, 'N', 'u', 'l', 'l', 's', 'o', 'f', 't', 'I', 'n', 's', 't'})
	binary.LittleEndian.PutUint32(data[64:], 1)
	binary.LittleEndian.PutUint32(data[84:], 128)
	if !nsisUninstallerHeader(data) {
		t.Fatal("NSIS uninstall header not recognized")
	}
	binary.LittleEndian.PutUint32(data[64:], 0)
	if nsisUninstallerHeader(data) {
		t.Fatal("installer accepted as uninstaller")
	}
	binary.LittleEndian.PutUint32(data[64:], 0x101)
	if nsisUninstallerHeader(data) {
		t.Fatal("invalid flags accepted")
	}
	if nsisUninstallerHeader(data[:72]) {
		t.Fatal("truncated header accepted")
	}
}
