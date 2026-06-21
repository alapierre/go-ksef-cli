package invoice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
)

type ExportCmd struct {
	Output         string        `arg:"" name:"output" help:"Output ZIP path" type:"path"`
	Token          string        `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier     string        `short:"i" required:"" help:"context identifier (NIP)"`
	SubjectType    string        `enum:"Subject1,Subject2,Subject3,SubjectAuthorized" default:"Subject1" help:"KSeF Subject type"`
	DateFrom       time.Time     `short:"f" required:"" help:"date from (RFC3339, e.g. 2026-06-01T00:00:00Z or 2026-06-01T00:00:00+02:00)"`
	DateTo         time.Time     `optional:"" help:"date to (RFC3339, e.g. 2026-06-30T23:59:59Z), default is now in UTC"`
	DateType       string        `enum:"Issue,Invoicing,PermanentStorage" default:"PermanentStorage" help:"Date type (Issue|Invoicing|PermanentStorage)"`
	Hwm            bool          `default:"false" help:"restrict to permanent storage high water mark date"`
	SelfInvoicing  bool          `help:"restrict to self-invoicing invoices"`
	FormType       string        `enum:"FA,PEF,FA_RR" default:"FA" help:"Schema form type (FA|PEF|FA_RR)"`
	OnlyMetadata   bool          `help:"export only invoice metadata"`
	PollInterval   time.Duration `default:"5s" help:"invoice export status polling interval"`
	WaitTimeout    time.Duration `default:"30m" help:"maximum time to wait for export package; use 0 for no timeout"`
	RequestTimeout time.Duration `default:"10m" help:"HTTP request timeout used by export operations"`
}

func (c *ExportCmd) Run(cfg *config.Config) error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be greater than 0")
	}

	appCtx, err := app.New(c.Token, cfg.Env, c.Identifier, app.WithTimeout(c.RequestTimeout))
	if err != nil {
		return err
	}
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)

	onlyMetadata := api.OptBool{}
	if c.OnlyMetadata {
		onlyMetadata = api.NewOptBool(true)
	}

	export, err := appCtx.Client.StartInvoiceExportWithGeneratedKey(ctx, c.filters(), onlyMetadata)
	if err != nil {
		return fmt.Errorf("start invoice export: %w", err)
	}

	fmt.Printf("Invoice export started. Reference number: %s\n", export.ReferenceNumber)

	status, err := c.waitForExport(ctx, appCtx.Client, export.ReferenceNumber)
	if err != nil {
		return err
	}

	if err := ensureParentDir(c.Output); err != nil {
		return err
	}

	out, err := os.Create(c.Output)
	if err != nil {
		return fmt.Errorf("create invoice export file: %w", err)
	}
	defer out.Close()

	result, err := appCtx.Client.DownloadInvoiceExport(ctx, status, export.Key, export.IV, out)
	if err != nil {
		return fmt.Errorf("download invoice export: %w", err)
	}

	fmt.Printf(
		"Invoice export saved to %s. Invoices: %d, bytes: %d, truncated: %t\n",
		c.Output,
		result.InvoiceCount,
		result.BytesWritten,
		result.IsTruncated,
	)
	return nil
}

func (c *ExportCmd) filters() api.InvoiceQueryFilters {
	dateTo := c.DateTo
	if dateTo.IsZero() {
		dateTo = time.Now().UTC()
	}

	filters := api.InvoiceQueryFilters{
		SubjectType: invoiceQuerySubjectType(c.SubjectType),
		DateRange: api.InvoiceQueryDateRange{
			DateType:                          invoiceQueryDateType(c.DateType),
			From:                              c.DateFrom,
			To:                                api.NewOptNilDateTime(dateTo),
			RestrictToPermanentStorageHwmDate: api.NewOptNilBool(c.Hwm),
		},
		FormType: api.NewOptNilInvoiceQueryFormType(api.InvoiceQueryFormType(c.FormType)),
	}
	if c.SelfInvoicing {
		filters.IsSelfInvoicing = api.NewOptNilBool(true)
	}

	return filters
}

func (c *ExportCmd) waitForExport(ctx context.Context, client *ksef.Client, referenceNumber string) (*api.InvoiceExportStatusResponse, error) {
	var deadline time.Time
	if c.WaitTimeout > 0 {
		deadline = time.Now().Add(c.WaitTimeout)
	}

	for {
		status, err := client.InvoiceExportStatus(ctx, referenceNumber)
		if err != nil {
			return nil, fmt.Errorf("check invoice export status: %w", err)
		}

		info := status.GetStatus()
		switch info.Code {
		case 200:
			fmt.Printf("Invoice export is ready. Status: %d %s\n", info.Code, info.Description)
			return status, nil
		case 100:
			fmt.Printf("Invoice export is being prepared. Status: %d %s\n", info.Code, info.Description)
		default:
			return nil, fmt.Errorf("invoice export failed with status %d: %s", info.Code, info.Description)
		}

		if !deadline.IsZero() && time.Now().Add(c.PollInterval).After(deadline) {
			return nil, fmt.Errorf("invoice export did not finish within %s", c.WaitTimeout)
		}

		timer := time.NewTimer(c.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func invoiceQuerySubjectType(value string) api.InvoiceQuerySubjectType {
	switch value {
	case "Subject2":
		return api.InvoiceQuerySubjectTypeSubject2
	case "Subject3":
		return api.InvoiceQuerySubjectTypeSubject3
	case "SubjectAuthorized":
		return api.InvoiceQuerySubjectTypeSubjectAuthorized
	default:
		return api.InvoiceQuerySubjectTypeSubject1
	}
}

func invoiceQueryDateType(value string) api.InvoiceQueryDateType {
	switch value {
	case "Issue":
		return api.InvoiceQueryDateTypeIssue
	case "Invoicing":
		return api.InvoiceQueryDateTypeInvoicing
	default:
		return api.InvoiceQueryDateTypePermanentStorage
	}
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create invoice export directory: %w", err)
	}
	return nil
}
