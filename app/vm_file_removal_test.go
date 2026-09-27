package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemovalObservationDoesNotAcceptRemainingPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.exe")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if awaitApplicationFilesRemoved(ctx, []string{path}, time.Millisecond) == nil {
		t.Fatal("remaining payload accepted")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("observer modified the payload")
	}
	done := make(chan error, 1)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	go func() { done <- awaitApplicationFilesRemoved(ctx2, []string{path}, time.Millisecond) }()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
