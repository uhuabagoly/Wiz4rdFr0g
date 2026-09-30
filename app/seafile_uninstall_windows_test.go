//go:build windows

package main

import (
	"bytes"
	"errors"
	"syscall"
	"testing"
)

type fakeSeafileSetting struct {
	value             seafileSetting
	readErr, writeErr error
}

func (s *fakeSeafileSetting) read() (seafileSetting, error) { return s.value, s.readErr }
func (s *fakeSeafileSetting) write(v seafileSetting) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.value = v
	return nil
}

func TestSeafileRetentionRestoresOriginalValue(t *testing.T) {
	for _, original := range []seafileSetting{{}, {present: true, kind: syscall.REG_DWORD, data: []byte{0, 0, 0, 0}}, {present: true, kind: syscall.REG_SZ, data: []byte{'0', 0, 0, 0}}} {
		store := &fakeSeafileSetting{value: original}
		restore, err := guardSeafileSettings(store)
		if err != nil || !bytes.Equal(store.value.data, seafileKeepSetting.data) || store.value.kind != syscall.REG_SZ {
			t.Fatalf("guard: %v %+v", err, store.value)
		}
		if err := restore(); err != nil {
			t.Fatal(err)
		}
		if store.value.present != original.present || store.value.kind != original.kind || !bytes.Equal(store.value.data, original.data) {
			t.Fatal("original setting not restored")
		}
	}
}

func TestSeafileRetentionFailsClosed(t *testing.T) {
	failure := errors.New("access denied")
	for _, store := range []*fakeSeafileSetting{{readErr: failure}, {writeErr: failure}} {
		if restore, err := guardSeafileSettings(store); err == nil || restore != nil {
			t.Fatal("failed setting operation accepted")
		}
	}
	store := &fakeSeafileSetting{}
	restore, err := guardSeafileSettings(store)
	if err != nil {
		t.Fatal(err)
	}
	store.value = seafileSetting{present: true, kind: syscall.REG_SZ, data: []byte{'2', 0, 0, 0}}
	if err := restore(); err == nil || store.value.data[0] != '2' {
		t.Fatal("concurrent setting was overwritten")
	}
}

func TestSeafileRegistrationBinding(t *testing.T) {
	const guid = "{812B8682-0C2A-4D5D-9EC6-97E3FC222955}"
	r := registryPackage{DisplayName: "Seafile 9.0.21", DisplayVersion: "9.0.21", WindowsInstaller: 1, Scope: "machine", RegistryView: "/reg:64", RegistryKey: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + guid, UninstallString: "MsiExec.exe /X" + guid}
	if !seafilePreserveDataRegistration(r) {
		t.Fatal("observed MSI rejected")
	}
	for _, mutate := range []func(*registryPackage){func(r *registryPackage) { r.DisplayVersion = "9.0.22" }, func(r *registryPackage) { r.RegistryView = "/reg:32" }, func(r *registryPackage) { r.Scope = "user" }, func(r *registryPackage) { r.RegistryKey += "-other" }, func(r *registryPackage) { r.UninstallString = "msiexec.exe /x {ECCD475D-04B5-490D-B5C1-BE60AE77183C}" }} {
		bad := r
		mutate(&bad)
		if seafilePreserveDataRegistration(bad) {
			t.Fatalf("unbound registration accepted: %+v", bad)
		}
	}
}
