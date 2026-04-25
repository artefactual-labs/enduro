package publisher

import (
	"errors"
	"fmt"
	"strings"
)

type Config struct {
	Type                  string
	Host                  string
	Port                  int
	User                  string
	Password              string
	PrivateKey            PrivateKeyConfig
	RemoteDir             string
	SubmittedPathPrefix   string
	HostKey               string
	KnownHostsFile        string
	InsecureIgnoreHostKey bool
}

type PrivateKeyConfig struct {
	Path       string
	Passphrase string
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Type) != ""
}

func (c Config) Validate() error {
	if !c.Enabled() {
		return nil
	}

	switch c.Type {
	case "sftp":
	default:
		return fmt.Errorf("invalid transfer publisher type %q", c.Type)
	}

	if strings.TrimSpace(c.Host) == "" {
		return errors.New("invalid transfer publisher configuration: host is required")
	}
	if strings.TrimSpace(c.User) == "" {
		return errors.New("invalid transfer publisher configuration: user is required")
	}
	if c.Port < 0 {
		return errors.New("invalid transfer publisher configuration: port must not be negative")
	}
	if strings.TrimSpace(c.Password) == "" && strings.TrimSpace(c.PrivateKey.Path) == "" {
		return errors.New("invalid transfer publisher configuration: password or privateKey.path is required")
	}
	if !c.InsecureIgnoreHostKey && strings.TrimSpace(c.HostKey) == "" && strings.TrimSpace(c.KnownHostsFile) == "" {
		return errors.New("invalid transfer publisher configuration: hostKey or knownHostsFile is required unless insecureIgnoreHostKey is enabled")
	}

	return nil
}
