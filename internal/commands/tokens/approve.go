package tokens

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/sirupsen/logrus"
)

type ApproveCmd struct {
	Identifier string `short:"i" required:"" help:"KSeF context NIP"`
	Reference  string `arg:"" name:"reference" help:"Reference number of the token awaiting approval"`
	Module     string `required:"" env:"KSEF_PKCS11_MODULE" type:"existingfile" help:"Path to the card vendor's PKCS#11 library"`
	Slot       int    `default:"0" env:"KSEF_PKCS11_SLOT" help:"Zero-based index among slots with a token present"`
}

func (c *ApproveCmd) Validate() error {
	if len(c.Identifier) != 10 {
		return fmt.Errorf("--identifier must be a ten-digit NIP")
	}
	for _, digit := range c.Identifier {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("--identifier must be a ten-digit NIP")
		}
	}
	if c.Slot < 0 {
		return fmt.Errorf("--slot must be non-negative")
	}
	return api.ReferenceNumber(c.Reference).Validate()
}

type approvalBearer string

func (b approvalBearer) Bearer(context.Context, api.OperationName) (api.Bearer, error) {
	return api.Bearer{Token: string(b)}, nil
}

func (c *ApproveCmd) Run(cfg *config.Config) error {
	var env ksef.Environment
	if err := env.UnmarshalText([]byte(cfg.Env)); err != nil {
		return err
	}
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, env)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	auth, err := ksef.NewAuthFacade(env, httpClient)
	if err != nil {
		return err
	}
	tokens, err := c.authenticate(ctx, auth)
	if err != nil {
		logrus.WithField("environment", env.Name()).WithError(err).Warn("Token approver authentication failed")
		return fmt.Errorf("authenticate approver: %w", err)
	}
	access := tokens.AccessToken.Token
	defer func() {
		if err := auth.CloseAuthSession(ctx, access); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not close the approver session: %v\n", err)
		}
	}()
	client, err := ksef.NewClient(env, httpClient, approvalBearer(access))
	if err != nil {
		return err
	}
	if err := client.ApproveKsefToken(ctx, c.Reference); err != nil {
		return fmt.Errorf("approve token %s: %w", c.Reference, err)
	}
	fmt.Printf("Token %s approved. Check its current status with tokens status.\n", c.Reference)
	return nil
}
