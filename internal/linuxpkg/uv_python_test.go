package linuxpkg

import "testing"

func TestManagedPythonInventoryRejectsSystemAndMalformedEvidence(t *testing.T) {
	key := "cpython-3.14.1-linux-x86_64-gnu"
	for _, bad := range []string{"null", "broken", "{}"} {
		if _, err := InventoryContains("uv-python", key, []byte(bad), 0); err == nil {
			t.Fatalf("accepted malformed inventory %s", bad)
		}
	}
	if present, err := InventoryContains("uv-python", key, []byte(`[{"key":"cpython-3.12.0-linux-x86_64-gnu","path":"/usr/bin/python3"}]`), 0); err != nil || present {
		t.Fatalf("wrong interpreter accepted: %v %v", present, err)
	}
	if present, err := InventoryContains("uv-python", key, []byte(`[{"key":"cpython-3.14.1-linux-x86_64-gnu","path":"/home/test/.local/share/uv/python/bin/python3"}]`), 0); err != nil || !present {
		t.Fatalf("exact interpreter not detected: %v %v", present, err)
	}
}
