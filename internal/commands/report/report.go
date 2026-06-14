package report

import (
	"fmt"

	"go-ksef-cli/internal/config"
	"go-ksef-cli/pkg/invoicereport"
)

type Cmd struct {
	Invoices InvoicesCmd `cmd:"invoices" help:"Create invoice CSV reports from invoice export ZIP"`
}

type InvoicesCmd struct {
	Input       string `arg:"" name:"zip" help:"Input invoice export ZIP path" type:"existingfile"`
	OutputDir   string `arg:"" name:"output-dir" help:"Output directory for CSV files" type:"path"`
	InvoicesCSV string `default:"invoices.csv" help:"Invoice metadata CSV file name"`
	RowsCSV     string `default:"invoice_rows.csv" help:"Invoice rows CSV file name"`
}

func (c *InvoicesCmd) Run(_ *config.Config) error {
	result, err := invoicereport.ExportInvoicesFromZIP(c.Input, c.OutputDir, invoicereport.Options{
		InvoicesCSVName: c.InvoicesCSV,
		RowsCSVName:     c.RowsCSV,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Invoice metadata CSV saved to %s. Invoices: %d\n", result.InvoicesCSVPath, result.InvoiceCount)
	fmt.Printf("Invoice rows CSV saved to %s. Rows: %d\n", result.RowsCSVPath, result.RowCount)
	return nil
}
