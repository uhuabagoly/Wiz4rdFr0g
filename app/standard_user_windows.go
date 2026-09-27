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
	"time"
	"unsafe"
)

// Start the ordinary user worker with a reduced token for the SAME user.
// This never requests credentials, changes accounts, or increases privilege.
func runStandardUserProcess(ctx context.Context, executable string, args []string) (int, string, error) {
	if !vmProcessElevated() {
		return runDirectProcess(ctx, executable, args)
	}
	if token, err := sameUserLinkedStandardToken(); err == nil {
		defer token.Close()
		workerLog("INFO", "Using the same user's verified non-elevated UAC linked token.")
		return runProcessWithUserToken(ctx, executable, args, token)
	} else {
		workerLog("INFO", "UAC linked token unavailable; using reduced token: "+err.Error())
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
	return runProcessWithUserToken(ctx, executable, args, token)
}

func sameUserLinkedStandardToken() (syscall.Token, error) {
	current, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return 0, err
	}
	defer current.Close()
	var linked syscall.Token
	var size uint32
	if err := syscall.GetTokenInformation(current, syscall.TokenLinkedToken, (*byte)(unsafe.Pointer(&linked)), uint32(unsafe.Sizeof(linked)), &size); err != nil {
		return 0, err
	}
	valid := false
	defer func() {
		if !valid {
			linked.Close()
		}
	}()
	var elevated uint32
	if err := syscall.GetTokenInformation(linked, syscall.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), 4, &size); err != nil || elevated != 0 {
		return 0, fmt.Errorf("linked token is not verified non-elevated: %v", err)
	}
	owner, err := current.GetTokenUser()
	if err != nil {
		return 0, err
	}
	other, err := linked.GetTokenUser()
	if err != nil {
		return 0, err
	}
	ownerSID, err := owner.User.Sid.String()
	if err != nil {
		return 0, err
	}
	otherSID, err := other.User.Sid.String()
	if err != nil || ownerSID == "" || ownerSID != otherSID {
		return 0, fmt.Errorf("linked token owner mismatch")
	}
	valid = true
	return linked, nil
}

func runProcessWithUserToken(ctx context.Context, executable string, args []string, token syscall.Token) (int, string, error) {
	command := exec.CommandContext(ctx, executable, args...)
	command.WaitDelay = 10 * time.Second
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
