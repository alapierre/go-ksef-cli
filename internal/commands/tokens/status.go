package tokens

import (
	"encoding/json"
	"os"

	"go-ksef-cli/internal/config"
)

type StatusCmd struct {
	Identifier string `short:"i" required:"" help:"Context identifier (NIP)"`
	Token      string `short:"t" env:"KSEF_TOKEN" help:"KSeF authorisation token for authentication"`
	Reference  string `arg:"" name:"reference" help:"Reference number of the token to check"`
}

func (c *StatusCmd) Run(cfg *config.Config) error {
	ctx, client, err := tokenClient(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}
	status, err := client.GetKsefTokenStatus(ctx, c.Reference)
	if err != nil {
		return err
	}
	output := json.NewEncoder(os.Stdout)
	output.SetIndent("", "  ")
	return output.Encode(status)
}
