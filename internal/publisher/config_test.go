package publisher

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg         Config
		errContains string
	}{
		"Defaults are safe": {
			cfg: Config{},
		},
		"SFTP accepts password authentication": {
			cfg: Config{
				Type:                  "sftp",
				Host:                  "ambox",
				User:                  "archivematica",
				Password:              "12345",
				InsecureIgnoreHostKey: true,
			},
		},
		"SFTP accepts private key authentication": {
			cfg: Config{
				Type: "sftp",
				Host: "ambox",
				User: "archivematica",
				PrivateKey: PrivateKeyConfig{
					Path: "/keys/id_ed25519",
				},
				InsecureIgnoreHostKey: true,
			},
		},
		"SFTP requires host": {
			cfg: Config{
				Type:                  "sftp",
				User:                  "archivematica",
				Password:              "12345",
				InsecureIgnoreHostKey: true,
			},
			errContains: "host is required",
		},
		"SFTP requires authentication": {
			cfg: Config{
				Type:                  "sftp",
				Host:                  "ambox",
				User:                  "archivematica",
				InsecureIgnoreHostKey: true,
			},
			errContains: "password or privateKey.path is required",
		},
		"SFTP requires host key policy": {
			cfg: Config{
				Type:     "sftp",
				Host:     "ambox",
				User:     "archivematica",
				Password: "12345",
			},
			errContains: "hostKey or knownHostsFile is required",
		},
		"Unsupported type is rejected": {
			cfg: Config{
				Type: "nfs",
			},
			errContains: `invalid transfer publisher type "nfs"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.errContains == "" {
				assert.NilError(t, err)
				return
			}

			assert.ErrorContains(t, err, tc.errContains)
		})
	}
}
