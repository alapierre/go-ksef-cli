package login

import (
	"context"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "command login")

type Cmd struct {
	Identifier        string `short:"i" help:"context identifier (NIP)"`
	Token             string `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	PrintSessionToken bool   `short:"p" env:"KSEF_PRINT_SESSION_TOKEN" help:"print session token"`
	NoStore           bool   `short:"n" help:"do not store session token"`
}

func (c *Cmd) Run(cfg *config.Config) error {

	fmt.Printf("Trying to login into KSeF, identifier: %s, env: %s\n", c.Identifier, cfg.Env)

	token, err := app.ResolveAuthToken(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}

	appContext, err := app.New(token, cfg.Env, c.Identifier, app.WithSessionPersistence(app.SessionPersistDisabled))
	if err != nil {
		return err
	}
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appContext.Env)
	ksefToken, err := ksef.WithKsefToken(ctx, appContext.AuthFacade, appContext.Encryptor, token)
	if err != nil {
		return err
	}

	if c.PrintSessionToken {
		app.PrintTokens(ksefToken)
	} else {
		fmt.Printf("login successful\n")
	}

	nip, ok := ksef.NipFromContext(ctx)
	if !ok {
		return fmt.Errorf("internal error: no NIP in context")
	}

	e, ok := ksef.EnvFromContext(ctx)
	if !ok {
		return fmt.Errorf("internal error: no env in context")
	}

	if !c.NoStore {
		if err := app.StoreSessionTokenWithError(ksefToken, e.Name(), nip); err != nil {
			return err
		}
	}

	return nil
}
