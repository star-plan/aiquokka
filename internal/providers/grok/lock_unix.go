//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package grok

import (
	"context"
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// acquireAuthLock uses the lock-file protocol used by the official Grok CLI.
// A non-blocking flock loop makes a cancelled request stop waiting promptly.
func acquireAuthLock(ctx context.Context, path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() {
				_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
				_ = f.Close()
			}, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			_ = f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}
