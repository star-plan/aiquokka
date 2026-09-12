//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package grok

import "context"

// Platforms without flock still use atomic writes. This fallback keeps the
// provider portable; supported Grok CLI platforms use lock_unix.go.
func acquireAuthLock(_ context.Context, _ string) (func(), error) { return func() {}, nil }
