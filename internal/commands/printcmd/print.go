package printcmd

import (
	"errors"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
)

type Cmd struct {
	Identifier string `short:"i" required:"" help:"context identifier (NIP)"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	t, err := app.LoadSessionToken(cfg.Env, c.Identifier)
	if err != nil {
		if errors.Is(err, app.ErrSessionTokenNotFound) {
			return fmt.Errorf("no stored session token for identifier %s and env %s; run: ksef-cli --env %s login --identifier %s", c.Identifier, cfg.Env, cfg.Env, c.Identifier)
		}
		return fmt.Errorf("cannot load stored session token: %w", err)
	}
	app.PrintTokens(t)
	return nil
}
