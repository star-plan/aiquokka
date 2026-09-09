//go:build darwin

package cursor

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const (
	keychainService = "cursor-access-token"
	keychainAccount = "cursor-user"
	keychainTimeout = 5 * time.Second
)

func loadOSStore() (*Credential, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keychainTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "security",
		"find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return nil, nil
	}
	return &Credential{AccessToken: token, Source: SourceOSStore}, nil
}
