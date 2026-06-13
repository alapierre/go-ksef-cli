package query

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/jedib0t/go-pretty/v6/table"

	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
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
		if err := exportCSV(c.Export, metadata); err != nil {
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

func exportCSV(path string, r *api.QueryInvoicesMetadataResponse) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create CSV export file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(invoiceMetadataCSVRecords(r)); err != nil {
		return fmt.Errorf("write CSV export file: %w", err)
	}

	return nil
}

func invoiceMetadataCSVRecords(r *api.QueryInvoicesMetadataResponse) [][]string {
	records := [][]string{{
		"ksef_number",
		"invoice_number",
		"issue_date",
		"invoicing_date",
		"acquisition_date",
		"permanent_storage_date",
		"seller_nip",
		"seller_name",
		"buyer_identifier_type",
		"buyer_identifier",
		"buyer_name",
		"net_amount",
		"gross_amount",
		"vat_amount",
		"currency",
		"invoicing_mode",
		"invoice_type",
		"form_system_code",
		"form_schema_version",
		"form_value",
		"is_self_invoicing",
		"has_attachment",
		"invoice_hash",
		"corrected_invoice_hash",
		"third_subjects",
		"authorized_subject_nip",
		"authorized_subject_name",
		"authorized_subject_role",
	}}

	for _, inv := range r.Invoices {
		records = append(records, []string{
			string(inv.KsefNumber),
			inv.InvoiceNumber,
			formatDate(inv.IssueDate),
			formatDateTime(inv.InvoicingDate),
			formatDateTime(inv.AcquisitionDate),
			formatDateTime(inv.PermanentStorageDate),
			string(inv.Seller.Nip),
			optString(inv.Seller.Name),
			string(inv.Buyer.Identifier.Type),
			optString(inv.Buyer.Identifier.Value),
			optString(inv.Buyer.Name),
			formatFloat(inv.NetAmount),
			formatFloat(inv.GrossAmount),
			formatFloat(inv.VatAmount),
			inv.Currency,
			string(inv.InvoicingMode),
			string(inv.InvoiceType),
			inv.FormCode.SystemCode,
			inv.FormCode.SchemaVersion,
			inv.FormCode.Value,
			strconv.FormatBool(inv.IsSelfInvoicing),
			strconv.FormatBool(inv.HasAttachment),
			base64.StdEncoding.EncodeToString(inv.InvoiceHash[:]),
			optHash(inv.HashOfCorrectedInvoice),
			thirdSubjects(inv.ThirdSubjects),
			authorizedSubjectNip(inv.AuthorizedSubject),
			authorizedSubjectName(inv.AuthorizedSubject),
			authorizedSubjectRole(inv.AuthorizedSubject),
		})
	}

	return records
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func optString(v api.OptNilString) string {
	if value, ok := v.Get(); ok {
		return value
	}
	return ""
}

func optHash(v api.OptNilSha256HashBase64) string {
	if value, ok := v.Get(); ok {
		return base64.StdEncoding.EncodeToString(value[:])
	}
	return ""
}

func thirdSubjects(v api.OptNilInvoiceMetadataThirdSubjectArray) string {
	subjects, ok := v.Get()
	if !ok || len(subjects) == 0 {
		return ""
	}

	type csvThirdSubject struct {
		Role            int32  `json:"role"`
		IdentifierType  string `json:"identifier_type"`
		IdentifierValue string `json:"identifier_value"`
		Name            string `json:"name"`
	}

	records := make([]csvThirdSubject, 0, len(subjects))
	for _, subject := range subjects {
		records = append(records, csvThirdSubject{
			Role:            subject.Role,
			IdentifierType:  string(subject.Identifier.Type),
			IdentifierValue: optString(subject.Identifier.Value),
			Name:            optString(subject.Name),
		})
	}

	data, err := json.Marshal(records)
	if err != nil {
		return ""
	}

	return string(data)
}

func authorizedSubjectNip(v api.OptNilInvoiceMetadataAuthorizedSubject) string {
	subject, ok := v.Get()
	if !ok {
		return ""
	}
	return string(subject.Nip)
}

func authorizedSubjectName(v api.OptNilInvoiceMetadataAuthorizedSubject) string {
	subject, ok := v.Get()
	if !ok {
		return ""
	}
	return optString(subject.Name)
}

func authorizedSubjectRole(v api.OptNilInvoiceMetadataAuthorizedSubject) string {
	subject, ok := v.Get()
	if !ok {
		return ""
	}
	return strconv.FormatInt(int64(subject.Role), 10)
}
