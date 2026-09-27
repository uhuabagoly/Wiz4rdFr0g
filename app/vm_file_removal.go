package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func awaitApplicationFilesRemoved(ctx context.Context, paths []string, interval time.Duration) error {
	for {
		remaining := ""
		for _, path := range paths {
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				continue
			}
			// Windows can briefly report access denied for a file pending
			// deletion. Wait for explicit not-exist; denial is never absence.
			if err != nil && !os.IsPermission(err) {
				return fmt.Errorf("application executable cannot be checked: %s: %w", path, err)
			}
			remaining = path
			break
		}
		if remaining == "" {
			return nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("application executable still exists or remains inaccessible: %s: %w", remaining, ctx.Err())
		case <-timer.C:
		}
	}
}
