package query

import (
	"context"
	"encoding/base64"
	"os"
	"time"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/jedib0t/go-pretty/v6/table"

	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"go-ksef-cli/pkg/invoicereport"
)

type Cmd struct {
	Token         string    `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier    string    `short:"i" required:"" help:"context identifier (NIP)"`
	SortOrder     string    `short:"s" enum:"Asc,Desc" default:"Asc" help:"sort order (Asc|Desc)"`
	PageOffset    int32     `short:"o" default:"0" help:"page offset"`
	PageSize      int32     `short:"p" default:"250" help:"page size (250 max)"`
	SubjectType   string    `enum:"Subject1,Subject2,Subject3,SubjectAuthorized" default:"Subject1" help:"KSeF Subject type"`
	DateFrom      time.Time `short:"f" required:"" help:"date from (yyyy-MM-ddTHH:mm:ss)"`
	DateTo        time.Time `optional:"" help:"date to (yyyy-MM-ddTHH:mm:ss), default is now in UTC"`
	DateType      string    `enum:"Issue,Invoicing,PermanentStorage" default:"PermanentStorage" help:"Date type (Issue|Invoicing|PermanentStorage)"`
	Hwm           bool      `default:"false" help:"restrict to permanent storage high water mark date"`
	SelfInvoicing bool      `help:"include or no self invoicing"`
	FormType      string    `enum:"FA,PEF,FA_RR" default:"FA" help:"Schema form type (FA|PEF|FA_RR)"`
	Export        string    `placeholder:"FILE" help:"export invoices metadata to CSV file" type:"path"`
}

func (c *Cmd) Run(cfg *config.Config) error {

	token, err := app.ResolveAuthToken(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}

	appCtx := app.New(token, cfg.Env)
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)

	metadata, err := appCtx.Client.QueryInvoicesMetadata(ctx, *c.filers(), *c.params())
	if err != nil {
		return err
	}

	if c.Export != "" {
		if err := invoicereport.WriteInvoiceMetadataCSV(c.Export, metadata); err != nil {
			return err
		}
	}

	printTable(metadata)

	return nil
}

func (c *Cmd) params() *api.InvoicesQueryMetadataPostParams {

	params := api.InvoicesQueryMetadataPostParams{
		SortOrder:  api.NewOptSortOrder(api.SortOrder(c.SortOrder)),
		PageOffset: api.NewOptInt32(c.PageOffset),
		PageSize:   api.NewOptInt32(c.PageSize),
	}

	return &params
}

func (c *Cmd) filers() *api.InvoiceQueryFilters {

	var subjectType api.InvoiceQuerySubjectType

	switch c.SubjectType {
	case "Subject1":
		subjectType = api.InvoiceQuerySubjectTypeSubject1
	case "Subject2":
		subjectType = api.InvoiceQuerySubjectTypeSubject2
	case "Subject3":
		subjectType = api.InvoiceQuerySubjectTypeSubject3
	case "SubjectAuthorized":
		subjectType = api.InvoiceQuerySubjectTypeSubjectAuthorized
	}

	var rangeType api.InvoiceQueryDateType
	switch c.DateType {
	case "Issue":
		rangeType = api.InvoiceQueryDateTypeIssue
	case "Invoicing":
		rangeType = api.InvoiceQueryDateTypeInvoicing
	case "PermanentStorage":
		rangeType = api.InvoiceQueryDateTypePermanentStorage
	}

	filters := api.InvoiceQueryFilters{
		SubjectType: subjectType,
		DateRange: api.InvoiceQueryDateRange{
			DateType:                          rangeType,
			From:                              c.DateFrom,
			To:                                api.NewOptNilDateTime(c.DateTo),
			RestrictToPermanentStorageHwmDate: api.NewOptNilBool(c.Hwm),
		},

		IsSelfInvoicing: api.NewOptNilBool(c.SelfInvoicing),
		FormType:        api.NewOptNilInvoiceQueryFormType(api.InvoiceQueryFormType(c.FormType)),
	}

	return &filters
}

func printTable(r *api.QueryInvoicesMetadataResponse) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"KSeF Number", "Invoice Number", "Issue Date", "Invoicing Date", "Seller", "Buyer", "Nett", "VAT", "Currency", "Invoice Hash"})

	for _, inv := range r.Invoices {
		t.AppendRows([]table.Row{
			{
				inv.KsefNumber,
				inv.InvoiceNumber,
				inv.IssueDate.Format("2006-01-02"),
				inv.InvoicingDate.Format(time.RFC3339),
				inv.Seller.Nip,
				inv.Buyer.Identifier.Value.Value,
				inv.NetAmount,
				inv.VatAmount,
				inv.Currency,
				base64.StdEncoding.EncodeToString(inv.InvoiceHash[:])},
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()

}
