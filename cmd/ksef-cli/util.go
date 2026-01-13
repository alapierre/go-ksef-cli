package main

import (
	"os"

	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func printInvoiceSendStatus(invoices []string) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Reference Number", "Timestamp", "Processing Code", "Processing Description"})

	for i, inv := range invoices {
		t.AppendRows([]table.Row{
			{i, inv, inv, inv, inv},
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}

func printTokens(r *api.AuthenticationTokensResponse) {

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
