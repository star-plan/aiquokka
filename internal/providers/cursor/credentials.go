package cursor

import "strings"

// CredentialSource names the store a token pair was read from. Access and
// refresh tokens from different sources must never be spliced together.
type CredentialSource string

const (
	SourceAgentFile CredentialSource = "agent-file"
	SourceOSStore   CredentialSource = "os-store"
	SourceDesktopDB CredentialSource = "desktop-db"
)

// Credential is one complete pair from a single source.
type Credential struct {
	AccessToken  string
	RefreshToken string
	Source       CredentialSource
}

var (
	loadAgentCredentials   = loadAgentAuthFile
	loadOSCredentials      = loadOSStore
	loadDesktopCredentials = loadDesktopDB
	discoverCredentials    = defaultDiscoverCredentials
)

func defaultDiscoverCredentials() ([]Credential, error) {
	var out []Credential
	add := func(c *Credential, err error) {
		if err != nil {
			debugf("skip source: %v", err)
			return
		}
		if c == nil || strings.TrimSpace(c.AccessToken) == "" {
			return
		}
		c.AccessToken = strings.TrimSpace(c.AccessToken)
		c.RefreshToken = strings.TrimSpace(c.RefreshToken)
		out = append(out, *c)
	}
	add(loadAgentCredentials())
	add(loadOSCredentials())
	add(loadDesktopCredentials())
	return out, nil
}
