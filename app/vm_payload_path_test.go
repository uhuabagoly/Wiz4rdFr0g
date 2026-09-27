package main

import "testing"

func TestApplicationPayloadExcludesMaintenance(t *testing.T) {
	for _, path := range []string{`C:\Users\runner\Package Cache\{id}\python-3.13.exe`, `C:\ProgramData\Package Cache`, `C:\App\unins000.exe`, `C:\App\Update.exe`, `C:\App\squirrel.exe`} {
		if vmApplicationPayloadPath(path) {
			t.Errorf("maintenance accepted: %s", path)
		}
	}
	if !vmApplicationPayloadPath(`C:\Python313\python.exe`) {
		t.Fatal("interpreter rejected")
	}
}
