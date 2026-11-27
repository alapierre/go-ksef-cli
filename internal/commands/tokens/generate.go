package tokens

import (
	"context"
	"fmt"
	"os"
	"time"

	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
)

type GenerateCmd struct {
	Plaintext   bool                      `help:"Write the token secret to --out in plaintext"`
	Out         string                    `required:"" type:"path" help:"New token file, encrypted by default with the local keystore key (see ksef-cli init)"`
	Identifier  string                    `short:"i" required:"" help:"Context identifier (NIP)"`
	Token       string                    `short:"t" env:"KSEF_TOKEN" help:"KSeF authorisation token for authentication"`
	Description string                    `required:"" help:"Description of the new token (5 to 256 characters)"`
	Permission  []api.TokenPermissionType `required:"" help:"Permission to grant, e.g. InvoiceRead. Repeat for additional permissions."`
	ValidFrom   *time.Time                `help:"Optional start of validity (RFC3339, including time zone)"`
	ValidTo     *time.Time                `help:"Optional end of validity (RFC3339, including time zone)"`
}

func (c *GenerateCmd) request() (api.TokensRev2PostReq, error) {
	req := api.TokensRev2PostReq{Permissions: c.Permission, Description: c.Description}
	if c.ValidFrom != nil {
		req.ValidFrom.SetTo(*c.ValidFrom)
	}
	if c.ValidTo != nil {
		req.ValidTo.SetTo(*c.ValidTo)
	}
	if c.ValidFrom != nil && c.ValidTo != nil && !c.ValidTo.After(*c.ValidFrom) {
		return req, fmt.Errorf("--valid-to must be later than --valid-from")
	}
	return req, req.Validate()
}

func (c *GenerateCmd) Run(cfg *config.Config) error {
	req, err := c.request()
	if err != nil {
		return err
	}
	ctx, client, err := tokenClient(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}
	issued, err := c.generateToFile(ctx, client, req, cfg.Env)
	if err != nil {
		return err
	}
	fmt.Printf("Reference number: %s\n", issued.ReferenceNumber)
	if issued.ExpiryDate != nil {
		fmt.Printf("Expiry date: %s\n", issued.ExpiryDate.Format(time.DateOnly))
	}
	fmt.Printf("Token saved to %s\n", c.Out)
	return nil
}

type tokenGenerator interface {
	GenerateKsefToken(context.Context, api.TokensRev2PostReq) (*ksef.GeneratedKsefToken, error)
}

// Reserve the destination before requesting the single-use response from KSeF.
func (c *GenerateCmd) generateToFile(ctx context.Context, client tokenGenerator, req api.TokensRev2PostReq, env string) (*ksef.GeneratedKsefToken, error) {
	var encrypt func([]byte) ([]byte, error)
	if !c.Plaintext {
		var err error
		encrypt, err = app.NewAuthTokenFileEncryptor(env, c.Identifier)
		if err != nil {
			return nil, err
		}
	}
	file, err := os.OpenFile(c.Out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create token file: %w", err)
	}
	issued, err := client.GenerateKsefToken(ctx, req)
	if err != nil {
		file.Close()
		os.Remove(c.Out)
		return nil, err
	}
	data := []byte(issued.Token)
	if encrypt != nil {
		data, err = encrypt(data)
	}
	if err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, fmt.Errorf("KSeF issued token %s but saving it to %s failed: %w", issued.ReferenceNumber, c.Out, err)
	}
	return issued, nil
}
