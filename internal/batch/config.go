package batch

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	BrowserRoot string
}

func (c *Config) Validate() error {
	return c.ValidateWithBaseDir("")
}

func (c *Config) ValidateWithBaseDir(baseDir string) error {
	if c.BrowserRoot == "" {
		return nil
	}

	root := c.BrowserRoot
	if baseDir != "" && !filepath.IsAbs(root) {
		root = filepath.Join(baseDir, root)
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("batch browser root: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("batch browser root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("batch browser root %q is not a directory", root)
	}

	c.BrowserRoot = root
	return nil
}
