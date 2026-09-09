package cursor

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	desktopAccessKey  = "cursorAuth/accessToken"
	desktopRefreshKey = "cursorAuth/refreshToken"
)

var desktopDBPath = defaultDesktopDBPath

func defaultDesktopDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		appData := strings.TrimSpace(os.Getenv("APPDATA"))
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb"), nil
	default:
		configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		capital := filepath.Join(configHome, "Cursor", "User", "globalStorage", "state.vscdb")
		lower := filepath.Join(configHome, "cursor", "User", "globalStorage", "state.vscdb")
		if _, err := os.Stat(capital); err == nil {
			return capital, nil
		}
		if _, err := os.Stat(lower); err == nil {
			return lower, nil
		}
		return capital, nil
	}
}

func loadDesktopDB() (*Credential, error) {
	path, err := desktopDBPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	db, err := openDesktopDB(path, true)
	if err != nil {
		return nil, fmt.Errorf("opening Cursor Desktop DB: %w", err)
	}
	defer db.Close()

	access, err := readItem(db, desktopAccessKey)
	if err != nil {
		return nil, err
	}
	refresh, err := readItem(db, desktopRefreshKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(access) == "" {
		return nil, nil
	}
	return &Credential{
		AccessToken:  access,
		RefreshToken: refresh,
		Source:       SourceDesktopDB,
	}, nil
}

func persistDesktopDB(c Credential) error {
	path, err := desktopDBPath()
	if err != nil {
		return err
	}
	db, err := openDesktopDB(path, false)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := upsertItem(db, desktopAccessKey, c.AccessToken); err != nil {
		return err
	}
	if c.RefreshToken != "" {
		if err := upsertItem(db, desktopRefreshKey, c.RefreshToken); err != nil {
			return err
		}
	}
	return nil
}

func openDesktopDB(path string, readonly bool) (*sql.DB, error) {
	dsn := "file:" + filepath.ToSlash(path)
	if readonly {
		dsn += "?mode=ro"
	}
	return sql.Open("sqlite", dsn)
}

func readItem(db *sql.DB, key string) (string, error) {
	var value []byte
	err := db.QueryRow(`SELECT value FROM ItemTable WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func upsertItem(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO ItemTable(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func writeDesktopTokens(path, access, refresh string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	db, err := openDesktopDB(path, false)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ItemTable (key TEXT PRIMARY KEY, value BLOB)`); err != nil {
		return err
	}
	if err := upsertItem(db, desktopAccessKey, access); err != nil {
		return err
	}
	return upsertItem(db, desktopRefreshKey, refresh)
}
