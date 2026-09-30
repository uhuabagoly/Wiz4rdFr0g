//go:build windows

package main

import (
	"bytes"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// The v9.0.21 MSI invokes the applet with --remove-user-data. Its documented
// HKCU setting skips both settings deletion and the blocking confirmation UI.
// https://help.seafile.com/faq/#preconfigure-options-for-windows-clients
func seafilePreserveDataRegistration(reg registryPackage) bool {
	const guid = "{812B8682-0C2A-4D5D-9EC6-97E3FC222955}"
	return reg.DisplayName == "Seafile 9.0.21" && reg.DisplayVersion == "9.0.21" && reg.Scope == "machine" && reg.RegistryView == "/reg:64" && reg.WindowsInstaller == 1 &&
		strings.EqualFold(reg.RegistryKey, `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+guid) &&
		strings.EqualFold(registeredMSIProductCode(reg.UninstallString, reg.QuietUninstallString, reg.RegistryKey, true), guid)
}

type seafileSetting struct {
	kind    uint32
	data    []byte
	present bool
}

type seafileSettingStore interface {
	read() (seafileSetting, error)
	write(seafileSetting) error
}

var seafileKeepSetting = seafileSetting{kind: syscall.REG_SZ, data: []byte{'1', 0, 0, 0}, present: true}

func guardSeafileSettings(store seafileSettingStore) (func() error, error) {
	before, err := store.read()
	if err != nil {
		return nil, err
	}
	if err := store.write(seafileKeepSetting); err != nil {
		return nil, err
	}
	return func() error {
		current, err := store.read()
		if err != nil {
			return err
		}
		if !current.present || current.kind != seafileKeepSetting.kind || !bytes.Equal(current.data, seafileKeepSetting.data) {
			return fmt.Errorf("Seafile retention setting changed during uninstall; refusing to overwrite it")
		}
		return store.write(before)
	}, nil
}

type seafileRegistrySetting struct{ key syscall.Handle }

func (s seafileRegistrySetting) read() (seafileSetting, error) {
	name := utf16Ptr("PreconfigureKeepConfigWhenUninstall")
	var kind, size uint32
	err := syscall.RegQueryValueEx(s.key, name, nil, &kind, nil, &size)
	if err == syscall.ERROR_FILE_NOT_FOUND {
		return seafileSetting{}, nil
	}
	if err != nil {
		return seafileSetting{}, err
	}
	if size > 65536 {
		return seafileSetting{}, fmt.Errorf("unexpectedly large Seafile retention setting")
	}
	data := make([]byte, size+1)
	if err := syscall.RegQueryValueEx(s.key, name, nil, &kind, &data[0], &size); err != nil {
		return seafileSetting{}, err
	}
	return seafileSetting{kind: kind, data: data[:size], present: true}, nil
}

func (s seafileRegistrySetting) write(value seafileSetting) error {
	dll := syscall.NewLazyDLL("advapi32.dll")
	name := utf16Ptr("PreconfigureKeepConfigWhenUninstall")
	var status uintptr
	if !value.present {
		status, _, _ = dll.NewProc("RegDeleteValueW").Call(uintptr(s.key), uintptr(unsafe.Pointer(name)))
		if status == uintptr(syscall.ERROR_FILE_NOT_FOUND) {
			return nil
		}
	} else {
		var data *byte
		if len(value.data) > 0 {
			data = &value.data[0]
		}
		status, _, _ = dll.NewProc("RegSetValueExW").Call(uintptr(s.key), uintptr(unsafe.Pointer(name)), 0, uintptr(value.kind), uintptr(unsafe.Pointer(data)), uintptr(len(value.data)))
	}
	if status != 0 {
		return syscall.Errno(status)
	}
	return nil
}

func prepareSeafileDataRetention() (func() error, error) {
	var key syscall.Handle
	status, _, _ := syscall.NewLazyDLL("advapi32.dll").NewProc("RegCreateKeyExW").Call(
		uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(utf16Ptr(`SOFTWARE\Seafile`))), 0, 0, 0,
		uintptr(syscall.KEY_QUERY_VALUE|syscall.KEY_SET_VALUE|syscall.KEY_WOW64_64KEY), 0, uintptr(unsafe.Pointer(&key)), 0)
	if status != 0 {
		return nil, syscall.Errno(status)
	}
	restore, err := guardSeafileSettings(seafileRegistrySetting{key: key})
	if err != nil {
		syscall.RegCloseKey(key)
		return nil, err
	}
	return func() error {
		defer syscall.RegCloseKey(key)
		// Restore only this value, preserving its original type and bytes. Never
		// remove the vendor key or any settings added by another process.
		return restore()
	}, nil
}
