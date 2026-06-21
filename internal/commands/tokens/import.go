package tokens

import (
	"errors"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"os"
)

type ImportCmd struct {
	Input    string `arg:"" name:"input" help:"input encrypted backup file" type:"existingfile"`
	Password string `short:"p" env:"KSEF_BACKUP_PASSWORD" optional:"" help:"backup password; if omitted, prompt securely"`
}

func (c *ImportCmd) Run(_ *config.Config) error {
	if !app.CheckIsEncKeyInitialized() {
		return fmt.Errorf("encryption key is not initialized; run: ksef-cli init")
	}

	password, err := resolvePassword(c.Password, false)
	if err != nil {
		return err
	}

	encrypted, err := os.ReadFile(c.Input)
	if err != nil {
		return fmt.Errorf("read encrypted token backup %s: %w", c.Input, err)
	}

	backup, err := app.DecryptTokenBackup(encrypted, password)
	if err != nil {
		if errors.Is(err, app.ErrInvalidTokenBackupPassword) {
			return fmt.Errorf("invalid token backup password")
		}
		return err
	}

	if err := app.ImportTokenBackup(backup); err != nil {
		return err
	}

	fmt.Printf("Token backup imported from %s. Entries: %d\n", c.Input, len(backup.Entries))
	return nil
}
