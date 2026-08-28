// Package zai reports Z.ai (GLM) usage bundles and cash balance.
package zai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/McKean/aiquokka/internal/usage"
)

// loadKey returns the Z.ai API key, trying $ZAI_API_KEY first and then the
// key stored by the pi coding agent (~/.pi/agent/models.json), which reads
// its Z.ai credentials the same way. Z.ai keys have the form "{id}.{secret}".
func loadKey() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ZAI_API_KEY")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err == nil {
		if key := piKey(filepath.Join(home, ".pi", "agent", "models.json")); key != "" {
			return key, nil
		}
	}
	return "", usage.NotConfigured("no Z.ai API key found — set ZAI_API_KEY ({id}.{secret}) or configure the zai provider in pi")
}

// piKey extracts providers.zai.apiKey from a pi models.json file, returning ""
// when the file or the entry is missing.
func piKey(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		Providers map[string]struct {
			APIKey string `json:"apiKey"`
		} `json:"providers"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Providers["zai"].APIKey)
}

// mintJWT signs the Zhipu-style authentication JWT: a HS256 token whose header
// carries sign_type "SIGN", whose payload carries the key id, an expiry and a
// timestamp, signed with the secret half of the API key. This is the only
// credential form the console endpoints accept — the raw API key is rejected.
func mintJWT(apiKey string) (string, error) {
	id, secret, ok := strings.Cut(apiKey, ".")
	if !ok || id == "" || secret == "" {
		return "", fmt.Errorf("Z.ai API key must have the form {id}.{secret}")
	}
	now := time.Now().UnixMilli()
	header := b64url([]byte(`{"alg":"HS256","sign_type":"SIGN"}`))
	payload, err := json.Marshal(map[string]any{
		"api_key":   id,
		"exp":       now + 3600_000,
		"timestamp": now,
	})
	if err != nil {
		return "", err
	}
	mac := hmacSign([]byte(secret), header+"."+b64url(payload))
	return header + "." + b64url(payload) + "." + mac, nil
}
