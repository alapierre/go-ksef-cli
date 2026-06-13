package initcmd

import (
	"fmt"
	"go-ksef-cli/internal/app"
)

type Cmd struct {
	ForceInit bool `help:"force initialization even if key is already initialized"`
}

func (c *Cmd) Run() error {

	if c.ForceInit || app.CheckIsEncKeyInitialized() {
		return fmt.Errorf("encryption key is already stored in keystore")
	}
	err := app.InitEncryption()
	if err != nil {
		return fmt.Errorf("cannot initialize encryption key: %w", err)
	}
	fmt.Println("Encryption key generated and stored")

	return nil
}
