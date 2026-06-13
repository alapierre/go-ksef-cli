package app

import (
	"os"

	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func PrintTokens(r *api.AuthenticationTokensResponse) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Type", "Value", "Valid until"})

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:           2,
			WidthMax:         100,
			WidthMaxEnforcer: text.WrapSoft,
		},
	})

	t.AppendRows([]table.Row{
		{"Access Token", r.AccessToken.Token, r.AccessToken.ValidUntil},
	})

	t.AppendRows([]table.Row{
		{"Refresh Token", r.RefreshToken.Token, r.RefreshToken.ValidUntil},
	})

	t.SetStyle(table.StyleLight)
	t.Render()
}
