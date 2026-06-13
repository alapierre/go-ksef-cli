package invoice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/cheggaaa/pb/v3"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "command invoice")

type Cmd struct {
	Download DownloadCmd `cmd:"" help:"Download invoice(s)"`
	PDF      PDFCmd      `cmd:"pdf" help:"Generate PDF visualizations for XML invoice files - it use ITrust KSeF PDF Service"`
}

type DownloadCmd struct {
	Path       string   `arg:"" name:"path" help:"Output path" type:"existingdir"`
	KsefNumber []string `short:"k" required:"" help:"Invoice KSeF number(s) to download"`
	Token      string   `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier string   `short:"i" required:"" help:"context identifier (NIP)"`
}

func (c *DownloadCmd) Run(cfg *config.Config) error {
	token, err := app.ResolveAuthToken(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}

	appCtx := app.New(token, cfg.Env)
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)

	results := make([]downloadResult, 0, len(c.KsefNumber))
	bar := pb.StartNew(len(c.KsefNumber))

	for _, ksefNumber := range c.KsefNumber {
		result := downloadInvoice(ctx, appCtx, c.Path, ksefNumber)
		results = append(results, result)
		bar.Increment()
	}

	bar.Finish()

	printDownloadSummary(results)

	failed := countFailedDownloads(results)
	if failed > 0 {
		return fmt.Errorf("failed to download %d of %d invoice(s)", failed, len(results))
	}

	return nil
}

type downloadResult struct {
	KsefNumber string
	Path       string
	Status     string
	Error      string
}

func downloadInvoice(ctx context.Context, appCtx *app.App, outputDir, ksefNumber string) downloadResult {
	result := downloadResult{
		KsefNumber: ksefNumber,
		Path:       invoiceXMLPath(outputDir, ksefNumber),
	}

	logger.Infof("Downloading invoice %s", ksefNumber)

	inv, err := appCtx.Client.GetInvoiceByKsefNumber(ctx, ksefNumber)
	if err != nil {
		logger.Warnf("Cannot download invoice %s: %v", ksefNumber, err)
		result.Status = "ERROR"
		result.Error = err.Error()
		return result
	}

	if err := saveInvoiceXML(result.Path, inv.Response); err != nil {
		logger.Warnf("Cannot save invoice %s to %s: %v", ksefNumber, result.Path, err)
		result.Status = "ERROR"
		result.Error = err.Error()
		return result
	}

	logger.Infof("Invoice %s saved to %s", ksefNumber, result.Path)
	result.Status = "OK"
	return result
}

func invoiceXMLPath(outputDir, ksefNumber string) string {
	return filepath.Join(outputDir, ksefNumber+".xml")
}

func saveInvoiceXML(path string, invoice io.Reader) error {
	if invoice == nil {
		return errors.New("empty invoice response")
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create invoice file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, invoice); err != nil {
		return fmt.Errorf("write invoice file: %w", err)
	}

	return nil
}

func countFailedDownloads(results []downloadResult) int {
	failed := 0
	for _, result := range results {
		if result.Status != "OK" {
			failed++
		}
	}
	return failed
}

func printDownloadSummary(results []downloadResult) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "KSeF Number", "Status", "Path", "Error"})

	for i, result := range results {
		t.AppendRows([]table.Row{
			{i + 1, result.KsefNumber, result.Status, result.Path, result.Error},
		})
	}

	t.SetStyle(table.StyleLight)
	t.Render()

	failed := countFailedDownloads(results)
	if failed > 0 {
		fmt.Printf("Downloaded %d invoice(s), failed %d of total: %d\n", len(results)-failed, failed, len(results))
		return
	}

	fmt.Printf("Successfully downloaded %d invoice(s)\n", len(results))
}
