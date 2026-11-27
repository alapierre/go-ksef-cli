package tokens

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

type ApproveCmd struct {
	Cert CertApproveCmd `cmd:"" help:"Sign with a KSeF Authentication certificate and its private key"`
	Card CardApproveCmd `cmd:"" default:"withargs" help:"Sign with a PKCS#11 smart card"`
}

type ApprovalTarget struct {
	Identifier string `short:"i" required:"" help:"KSeF context NIP"`
	Reference  string `arg:"" name:"reference" help:"Reference number of the token awaiting approval"`
}

type CardApproveCmd struct {
	ApprovalTarget
	Module string `required:"" env:"KSEF_PKCS11_MODULE" type:"existingfile" help:"Path to the card vendor's PKCS#11 library"`
	Slot   int    `default:"0" env:"KSEF_PKCS11_SLOT" help:"Zero-based index among slots with a token present"`
}

type CertApproveCmd struct {
	ApprovalTarget
	Cert string `short:"c" required:"" type:"existingfile" help:"KSeF Authentication certificate in PEM format"`
	Key  string `short:"k" required:"" type:"existingfile" help:"Encrypted PKCS#8 PEM private key (password from terminal or KSEF_KEY_PASSWORD)"`
}

func (c *ApprovalTarget) Validate() error {
	if len(c.Identifier) != 10 {
		return fmt.Errorf("--identifier must be a ten-digit NIP")
	}
	for _, digit := range c.Identifier {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("--identifier must be a ten-digit NIP")
		}
	}
	return api.ReferenceNumber(c.Reference).Validate()
}

func (c *CardApproveCmd) Validate() error {
	if c.Slot < 0 {
		return fmt.Errorf("--slot must be non-negative")
	}
	return c.ApprovalTarget.Validate()
}

type approvalBearer string

func (b approvalBearer) Bearer(context.Context, api.OperationName) (api.Bearer, error) {
	return api.Bearer{Token: string(b)}, nil
}

func (c *CardApproveCmd) Run(cfg *config.Config) error {
	return c.approve(cfg, []string{"card", "-d", c.Module, "-s", strconv.Itoa(c.Slot)}, "")
}

func (c *CertApproveCmd) Run(cfg *config.Config) error {
	password := os.Getenv("KSEF_KEY_PASSWORD")
	if password == "" {
		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			return fmt.Errorf("set KSEF_KEY_PASSWORD or run in a terminal to enter the private key password")
		}
		fmt.Fprint(os.Stderr, "Private key password: ")
		value, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("read private key password: %w", err)
		}
		if len(value) == 0 {
			return fmt.Errorf("private key password cannot be empty")
		}
		password = string(value)
		clear(value)
	}
	return c.approve(cfg, []string{"keystore", "-c", c.Cert, "-k", c.Key}, password)
}

func (c *ApprovalTarget) approve(cfg *config.Config, signerArgs []string, password string) error {
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
	tokens, err := c.authenticate(ctx, auth, signerArgs, password)
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
