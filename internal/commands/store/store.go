package store

import (
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
)

type Cmd struct {
	Identifier string `short:"i" required:"" help:"context identifier (NIP)"`
	Token      string `short:"t" env:"KSEF_TOKEN" required:"" help:"KSeF authorisation token to store"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	app.StoreAuthToken([]byte(c.Token), cfg.Env, c.Identifier)
	fmt.Printf("Token for identifier %s and environment: %s stored successfully\n", c.Identifier, cfg.Env)
	return nil
}
