package store

import (
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
)

type Cmd struct {
	EncryptedTokenFile string `type:"existingfile" help:"Store the token from a keystore-encrypted file (takes precedence over --token and KSEF_TOKEN)"`
	Identifier         string `short:"i" required:"" help:"context identifier (NIP)"`
	Token              string `short:"t" env:"KSEF_TOKEN" help:"KSeF authorisation token to store"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	token := c.Token
	if c.EncryptedTokenFile != "" {
		var err error
		token, err = app.ReadEncryptedAuthTokenFile(c.EncryptedTokenFile, cfg.Env, c.Identifier)
		if err != nil {
			return err
		}
	}
	if token == "" {
		return fmt.Errorf("provide --encrypted-token-file, --token or KSEF_TOKEN")
	}
	if err := app.StoreAuthToken([]byte(token), cfg.Env, c.Identifier); err != nil {
		return fmt.Errorf("cannot store KSeF authorisation token: %w", err)
	}
	fmt.Printf("Token for identifier %s and environment: %s stored successfully\n", c.Identifier, cfg.Env)
	return nil
}
