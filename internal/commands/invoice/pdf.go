package invoice

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/sirupsen/logrus"
)

type PDFCmd struct {
	URL      string        `name:"url" short:"u" required:"" help:"ITrust KSeF Visualization service URL, e.g. http://localhost:8080/invoice/pdf."`
	Login    string        `name:"login" short:"l" env:"ITRUST_VS_LOGIN" required:"" help:"Basic auth login."`
	Password string        `name:"password" short:"p" env:"ITRUST_VS_PASSWORD" required:"" help:"Basic auth password."`
	Source   string        `name:"source" arg:"" required:"" type:"existingdir" help:"Directory with source XML files."`
	Output   string        `name:"output" short:"o" type:"path" help:"Directory for generated PDF files. Defaults to source directory."`
	Timeout  time.Duration `name:"timeout" default:"60s" help:"HTTP request timeout."`
}

func (c *PDFCmd) Run() error {
	sourceDir, err := filepath.Abs(c.Source)
	if err != nil {
		return fmt.Errorf("cannot resolve source path: %w", err)
	}

	outputDir := c.Output
	if outputDir == "" {
		outputDir = sourceDir
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("cannot resolve output path: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("cannot create output directory: %w", err)
	}

	xmlFiles, err := findXMLFiles(sourceDir)
	if err != nil {
		return err
	}
	if len(xmlFiles) == 0 {
		fmt.Println("No XML files found.")
		logger.WithField("source_dir", sourceDir).Info("no XML files found for PDF generation")
		return nil
	}

	logger.WithFields(logrus.Fields{
		"source_dir": sourceDir,
		"output_dir": outputDir,
		"file_count": len(xmlFiles),
		"url":        c.URL,
		"timeout":    c.Timeout.String(),
	}).Info("starting PDF generation")

	client := &http.Client{Timeout: c.Timeout}
	results := make([]pdfResult, 0, len(xmlFiles))
	bar := pb.New(len(xmlFiles)).SetWriter(os.Stderr).Start()

	for _, xmlPath := range xmlFiles {
		result := processXML(client, c, xmlPath, outputDir)
		results = append(results, result)
		bar.Increment()
	}

	bar.Finish()
	printPDFSummary(results)

	failed := countFailedPDFResults(results)
	if failed > 0 {
		return fmt.Errorf("failed to generate %d of %d PDF file(s); see application log for details", failed, len(results))
	}

	logger.Info("PDF generation finished")
	return nil
}

type pdfResult struct {
	XMLPath string
	PDFPath string
	Status  string
	Error   string
}

func findXMLFiles(sourceDir string) ([]string, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read source directory: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			files = append(files, filepath.Join(sourceDir, entry.Name()))
		}
	}

	return files, nil
}

func processXML(client *http.Client, cmd *PDFCmd, xmlPath, outputDir string) pdfResult {
	result := pdfResult{
		XMLPath: xmlPath,
		PDFPath: pdfPathFor(xmlPath, outputDir),
	}

	logEntry := logger.WithField("xml_path", xmlPath)

	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		return failedPDFResult(result, logEntry, fmt.Errorf("cannot open XML: %w", err))
	}
	defer xmlFile.Close()

	request, err := http.NewRequest(http.MethodPost, cmd.URL, xmlFile)
	if err != nil {
		return failedPDFResult(result, logEntry, fmt.Errorf("cannot prepare request: %w", err))
	}
	request.Header.Set("Content-Type", "application/xml")
	request.SetBasicAuth(cmd.Login, cmd.Password)

	response, err := client.Do(request)
	if err != nil {
		return failedPDFResult(result, logEntry, fmt.Errorf("visualization service request failed: %w", err))
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return failedPDFResult(result, logEntry, fmt.Errorf("visualization service returned %s: %s", response.Status, strings.TrimSpace(string(body))))
	}

	outputFile, err := os.Create(result.PDFPath)
	if err != nil {
		return failedPDFResult(result, logEntry, fmt.Errorf("cannot create PDF file: %w", err))
	}
	defer outputFile.Close()

	if _, err := io.Copy(outputFile, response.Body); err != nil {
		return failedPDFResult(result, logEntry, fmt.Errorf("cannot write PDF response: %w", err))
	}

	logEntry.WithField("pdf_path", result.PDFPath).Info("PDF generated")
	result.Status = "OK"
	return result
}

func failedPDFResult(result pdfResult, logEntry *logrus.Entry, err error) pdfResult {
	logEntry.WithError(err).Error("failed to generate PDF")
	result.Status = "ERROR"
	result.Error = err.Error()
	return result
}

func pdfPathFor(xmlPath, outputDir string) string {
	baseName := filepath.Base(xmlPath)
	extension := filepath.Ext(baseName)
	pdfName := strings.TrimSuffix(baseName, extension) + ".pdf"
	return filepath.Join(outputDir, pdfName)
}

func countFailedPDFResults(results []pdfResult) int {
	failed := 0
	for _, result := range results {
		if result.Status != "OK" {
			failed++
		}
	}
	return failed
}

func printPDFSummary(results []pdfResult) {
	failed := countFailedPDFResults(results)
	if failed > 0 {
		fmt.Printf("Generated %d PDF file(s), failed %d of total: %d\n", len(results)-failed, failed, len(results))
		return
	}

	fmt.Printf("Generated %d PDF file(s).\n", len(results))
}
