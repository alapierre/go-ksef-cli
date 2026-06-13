package session

import (
	"context"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
)

type CmdSessionClose struct {
	Token      string `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier string `short:"i" required:"" help:"context identifier (NIP)"`
	SessionId  string `short:"s" required:"" help:"session identifier"`
}

func (c *CmdSessionClose) Run(cfg *config.Config) error {

	token, err := app.ResolveAuthToken(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}

	appCtx := app.New(token, cfg.Env)
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)

	session, err := appCtx.Client.CloseInteractiveSession(ctx, c.SessionId)
	if err != nil {
		return err
	}

	fmt.Println(session)
	return nil
}
