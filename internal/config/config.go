// Package config loads ztpanel's runtime configuration from environment
// variables, with sane defaults for a typical single-host ZeroTier
// controller setup.
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	// ListenAddr is the address ztpanel's HTTP server binds to.
	ListenAddr string

	// ZTControllerURL is the base URL of the local ZeroTier controller
	// JSON API (the zerotier-one daemon itself, not ztpanel).
	ZTControllerURL string

	// ZTAuthTokenPath is the file holding the ZeroTier controller's
	// local auth token (authtoken.secret).
	ZTAuthTokenPath string

	// DataDir holds ztpanel's own state: the user store and the
	// session-signing key. Kept separate from ZeroTier's own data dir.
	DataDir string
}

func Load() Config {
	cfg := Config{
		ListenAddr:      getenv("ZTPANEL_LISTEN", ":8787"),
		ZTControllerURL: getenv("ZTPANEL_ZT_URL", "http://127.0.0.1:9993"),
		ZTAuthTokenPath: getenv("ZTPANEL_ZT_AUTHTOKEN", "/var/lib/zerotier-one/authtoken.secret"),
		DataDir:         getenv("ZTPANEL_DATA_DIR", "./data"),
	}
	return cfg
}

func (c Config) UsersFile() string {
	return filepath.Join(c.DataDir, "users.json")
}

func (c Config) SessionKeyFile() string {
	return filepath.Join(c.DataDir, "session.key")
}

func (c Config) OrganizationsFile() string {
	return filepath.Join(c.DataDir, "organizations.json")
}

func (c Config) ActivityLogFile() string {
	return filepath.Join(c.DataDir, "activity.jsonl")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func EnsureDataDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}
