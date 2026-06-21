package logout

import (
	"context"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
)

type Cmd struct {
	Token      string `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier string `short:"i" required:"" help:"context identifier (NIP)"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	appCtx, err := app.New(c.Token, cfg.Env, c.Identifier, app.WithSessionPersistence(app.SessionPersistDisabled))
	if err != nil {
		return err
	}

	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)
	bearer, err := appCtx.TokenProvider.Bearer(ctx, api.AuthSessionsCurrentDeleteOperation)
	if err != nil {
		return fmt.Errorf("resolve token for logout: %w", err)
	}

	if err := appCtx.AuthFacade.CloseAuthSession(ctx, bearer.Token); err != nil {
		return fmt.Errorf("close auth session in KSeF: %w", err)
	}

	if err := app.DeleteSessionToken(cfg.Env, c.Identifier); err != nil {
		return fmt.Errorf("remove stored session token: %w", err)
	}

	_ = appCtx.TokenProvider.Invalidate(ctx)
	fmt.Printf("logout successful\n")
	return nil
}
