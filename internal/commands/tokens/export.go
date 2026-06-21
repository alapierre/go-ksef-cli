package tokens

import (
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"os"
	"path/filepath"
	"strings"
)

type ExportCmd struct {
	Output     string `arg:"" name:"output" help:"output encrypted backup file" type:"path"`
	Identifier string `short:"i" help:"context identifier (NIP); when set export only this identifier"`
	All        bool   `help:"export tokens from all environments"`
	Password   string `short:"p" env:"KSEF_BACKUP_PASSWORD" optional:"" help:"backup password; if omitted, prompt securely"`
}

func (c *ExportCmd) Run(cfg *config.Config) error {
	password, err := resolvePassword(c.Password, true)
	if err != nil {
		return err
	}

	backup, warnings, err := app.BuildTokenBackupBestEffort(cfg.Env, c.All, c.Identifier)
	if err != nil {
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}
		return err
	}

	encrypted, err := app.EncryptTokenBackup(backup, password)
	if err != nil {
		return err
	}

	if err := ensureParentDir(c.Output); err != nil {
		return err
	}
	if err := os.WriteFile(c.Output, encrypted, 0600); err != nil {
		return fmt.Errorf("write encrypted token backup %s: %w", c.Output, err)
	}

	fmt.Printf("Token backup exported to %s. Entries: %d\n", c.Output, len(backup.Entries))
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
	return nil
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || strings.TrimSpace(dir) == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	return nil
}
