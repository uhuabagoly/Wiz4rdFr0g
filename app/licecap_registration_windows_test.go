//go:build windows

package main

import (
	"path/filepath"
	"testing"
)

func TestLICEcapRootRequiresPublisherMachineDirectory(t *testing.T) {
	pf := t.TempDir()
	if !licecapInstallRoot(filepath.Join(pf, "LICEcap"), pf) {
		t.Fatal("publisher default rejected")
	}
	for _, root := range []string{pf, "LICEcap", filepath.Join(pf, "Other"), filepath.Join(pf, "LICEcap", "child"), filepath.Join(pf, "LICEcap-old")} {
		if licecapInstallRoot(root, pf) {
			t.Fatalf("unbound path accepted: %s", root)
		}
	}
	if licecapInstallRoot(filepath.Join(pf, "LICEcap"), "") {
		t.Fatal("missing protected root accepted")
	}
}
