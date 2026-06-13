package session

import (
	"context"
	"encoding/base64"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"os"
	"time"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "command session")

type Cmd struct {
	Invoices CmdSessionInvoices `cmd:"" help:"List invoices for the current session."`
	Close    CmdSessionClose    `cmd:"" help:"Close interactive session"`
}

type CmdSessionInvoices struct {
	Token      string `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier string `short:"i" required:"" help:"context identifier (NIP)"`
	SessionId  string `short:"s" required:"" help:"session identifier"`
	PageSize   int32  `short:"p" default:"1000" help:"page size (1000 max)"`
}

func (c *CmdSessionInvoices) Run(cfg *config.Config) error {

	token, err := app.ResolveAuthToken(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}

	appCtx := app.New(token, cfg.Env)
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)

	continuationToken := api.OptString{}
	page := 1
	total := 0

	for {
		invoices, err := appCtx.Client.SessionInvoices(ctx, c.SessionId, continuationToken, api.NewOptInt32(c.PageSize))
		if err != nil {
			return err
		}

		total += len(invoices.Invoices)
		fmt.Printf("invoices for session %s, page: %d, number of invoices: %d, total printed: %d\n", c.SessionId, page, len(invoices.Invoices), total)
		printTable(invoices)

		nextToken, ok := invoices.ContinuationToken.Get()
		if !ok || nextToken == "" {
			break
		}

		continuationToken = api.NewOptString(nextToken)
		page++
	}

	return nil
}

func printTable(r *api.SessionInvoicesResponse) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Reference Number", "Status", "Description", "Invoice Number", "KSeF Number", "Acquisition Date", "Invoicing Date", "Permanent Storage Date", "invoice Hash"})

	for _, inv := range r.Invoices {
		t.AppendRows([]table.Row{
			{
				inv.OrdinalNumber,
				inv.ReferenceNumber,
				inv.Status.Code,
				inv.Status.Description,
				formatOptNilString(inv.InvoiceNumber),
				formatOptNilKSeFNumber(inv.KsefNumber),
				formatOptNilDateTime(inv.AcquisitionDate),
				inv.InvoicingDate.Format(time.RFC3339),
				formatOptNilDateTime(inv.PermanentStorageDate),
				base64.StdEncoding.EncodeToString(inv.InvoiceHash[:]),
			},
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}

func formatOptNilDateTime(d api.OptNilDateTime) string {

	if !d.IsSet() || d.IsNull() {
		return "-"
	}

	return d.Value.Format(time.RFC3339)
}

func formatOptNilString(s api.OptNilString) string {
	if s.IsNull() {
		return "-"
	}

	return s.Value
}

func formatOptNilKSeFNumber(s api.OptNilKsefNumber) string {
	if !s.IsSet() || s.IsNull() {
		return "-"
	}

	return string(s.Value)
}

func formatOprUrl(r api.OptNilURI) string {
	if !r.IsSet() || r.IsNull() {
		return "-"
	}
	return r.Value.String()
}
