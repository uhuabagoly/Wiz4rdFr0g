//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

// Start the ordinary user worker with a reduced token for the SAME user.
// This never requests credentials, changes accounts, or increases privilege.
func runStandardUserProcess(ctx context.Context, executable string, args []string) (int, string, error) {
	if !vmProcessElevated() {
		return runDirectProcess(ctx, executable, args)
	}
	api := syscall.NewLazyDLL("advapi32.dll")
	create := api.NewProc("SaferCreateLevel")
	compute := api.NewProc("SaferComputeTokenFromLevel")
	closeLevel := api.NewProc("SaferCloseLevel")
	setInformation := api.NewProc("SetTokenInformation")
	for _, proc := range []*syscall.LazyProc{create, compute, closeLevel, setInformation} {
		if err := proc.Find(); err != nil {
			return -1, "", err
		}
	}
	var level syscall.Handle
	ok, _, err := create.Call(2, 0x20000, 1, uintptr(unsafe.Pointer(&level)), 0)
	if ok == 0 {
		return -1, "", fmt.Errorf("create standard-user security level: %w", err)
	}
	defer closeLevel.Call(uintptr(level))
	var token syscall.Token
	ok, _, err = compute.Call(uintptr(level), 0, uintptr(unsafe.Pointer(&token)), 0, 0)
	if ok == 0 {
		return -1, "", fmt.Errorf("reduce current user's token: %w", err)
	}
	defer token.Close()
	sid, err := syscall.StringToSid("S-1-16-8192") // medium integrity
	if err != nil {
		return -1, "", err
	}
	label := struct{ Label syscall.SIDAndAttributes }{syscall.SIDAndAttributes{Sid: sid, Attributes: 0x20}}
	ok, _, err = setInformation.Call(uintptr(token), 25, uintptr(unsafe.Pointer(&label)), unsafe.Sizeof(label)+uintptr(sid.Len()))
	runtime.KeepAlive(sid)
	if ok == 0 {
		return -1, "", fmt.Errorf("set standard-user integrity: %w", err)
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, Token: token}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "WIZ4RDFR0G_EVIDENCE_HMAC_KEY=") {
			command.Env = append(command.Env, entry)
		}
	}
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return -1, string(output), ctx.Err()
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), string(output), err
		}
		return -1, string(output), err
	}
	return 0, string(output), nil
}
