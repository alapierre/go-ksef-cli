package status

import (
	"errors"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type Cmd struct {
	All bool `help:"show token status for all environments"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	statuses, err := app.ListTokenStatuses(cfg.Env, c.All)
	if err != nil {
		return err
	}

	if len(statuses) == 0 {
		if c.All {
			fmt.Println("No stored KSeF tokens found.")
			return nil
		}
		fmt.Printf("No stored KSeF tokens found for env %s.\n", strings.ToUpper(cfg.Env))
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"Env",
		"Identifier",
		"Auth Token",
		"Session Pair",
		"Access Valid",
		"Access Exp",
		"Refresh Valid",
		"Refresh Exp",
		"Logged In",
		"Errors",
	})
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:             "Errors",
			WidthMax:         48,
			WidthMaxEnforcer: text.WrapHard,
		},
	})

	for _, s := range statuses {
		t.AppendRow(table.Row{
			s.Env,
			s.Identifier,
			authTokenCell(s),
			sessionPairCell(s),
			boolCell(s.AccessValid, s.HasSessionTokens && s.SessionTokenErr == nil),
			timeCell(s.AccessValidUntil),
			boolCell(s.RefreshValid, s.HasSessionTokens && s.SessionTokenErr == nil),
			timeCell(s.RefreshValidUntil),
			boolCell(s.IsLoggedIn(), s.HasSessionTokens && s.SessionTokenErr == nil),
			errorCell(s),
		})
	}

	t.SetStyle(table.StyleLight)
	t.Render()
	return nil
}

func authTokenCell(s app.TokenStorageStatus) string {
	if !s.HasAuthToken {
		return "no"
	}
	if s.AuthTokenErr != nil {
		return "error"
	}
	return "yes"
}

func sessionPairCell(s app.TokenStorageStatus) string {
	if !s.HasSessionTokens {
		return "no"
	}
	if s.SessionTokenErr != nil {
		return "error"
	}
	return "yes"
}

func boolCell(value bool, defined bool) string {
	if !defined {
		return "-"
	}
	if value {
		return "yes"
	}
	return "no"
}

func timeCell(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.In(time.Local).Format("2006-01-02 15:04:05 -07:00 MST")
}

func errorCell(s app.TokenStorageStatus) string {
	msgs := make([]string, 0, 2)
	if s.AuthTokenErr != nil {
		msgs = append(msgs, "auth: "+authErrorHint(s))
	}
	if s.SessionTokenErr != nil {
		msgs = append(msgs, "session: "+s.SessionTokenErr.Error())
	}
	return strings.Join(msgs, " | ")
}

func authErrorHint(s app.TokenStorageStatus) string {
	if errors.Is(s.AuthTokenErr, app.ErrAuthTokenDecrypt) {
		if s.IsLoggedIn() {
			return fmt.Sprintf("stored auth token cannot be decrypted (current session still works); run: ksef-cli --env %s store --identifier %s --token \"...\"", s.Env, s.Identifier)
		}
		return fmt.Sprintf("stored auth token cannot be decrypted; run: ksef-cli --env %s store --identifier %s --token \"...\"", s.Env, s.Identifier)
	}
	return s.AuthTokenErr.Error()
}
