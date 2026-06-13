package printcmd

import (
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
)

type Cmd struct {
	Identifier string `short:"i" required:"" help:"context identifier (NIP)"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	t := app.LoadSessionToken(cfg.Env, c.Identifier)
	app.PrintTokens(t)
	return nil
}
