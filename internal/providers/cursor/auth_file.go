package cursor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/star-plan/aiquokka/internal/credential"
)

var agentAuthPath = defaultAgentAuthPath

type agentAuthFile struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func defaultAgentAuthPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := strings.TrimSpace(os.Getenv("APPDATA"))
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "auth.json"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".cursor", "auth.json"), nil
	default:
		if configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); configHome != "" {
			return filepath.Join(configHome, "cursor", "auth.json"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "cursor", "auth.json"), nil
	}
}

func loadAgentAuthFile() (*Credential, error) {
	path, err := agentAuthPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var file agentAuthFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if strings.TrimSpace(file.AccessToken) == "" {
		return nil, nil
	}
	return &Credential{
		AccessToken:  file.AccessToken,
		RefreshToken: file.RefreshToken,
		Source:       SourceAgentFile,
	}, nil
}

func persistAgentAuthFile(c Credential) error {
	path, err := agentAuthPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	set := func(k, v string) {
		if b, err := json.Marshal(v); err == nil {
			raw[k] = b
		}
	}
	set("accessToken", c.AccessToken)
	if c.RefreshToken != "" {
		set("refreshToken", c.RefreshToken)
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return credential.WriteFileAtomic(path, out, 0o600)
}
