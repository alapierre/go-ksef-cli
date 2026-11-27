package tokens

import (
	"context"
	"go-ksef-cli/internal/app"

	"github.com/alapierre/go-ksef-client/ksef"
)

type Cmd struct {
	Approve  ApproveCmd  `cmd:"" help:"Approve a KSeF token using a separate XAdES session"`
	Generate GenerateCmd `cmd:"" help:"Generate a KSeF token (rev2)"`
	Status   StatusCmd   `cmd:"" help:"Get the current KSeF token status by reference number (rev2)"`
	Export   ExportCmd   `cmd:"" help:"Export stored KSeF tokens to encrypted backup"`
	Import   ImportCmd   `cmd:"" help:"Import stored KSeF tokens from encrypted backup"`
}

func tokenClient(token, env, identifier string) (context.Context, *ksef.Client, error) {
	mode := app.SessionPersistAuto
	if token != "" {
		mode = app.SessionIsolated
	}
	a, err := app.New(token, env, identifier, app.WithSessionPersistence(mode))
	if err != nil {
		return nil, nil, err
	}
	return ksef.ContextWithEnv(context.Background(), identifier, a.Env), a.Client, nil
}
