package invoicereport

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	DefaultMetadataFileName = "_metadata.json"
	DefaultInvoicesCSVName  = "invoices.csv"
	DefaultRowsCSVName      = "invoice_rows.csv"
)

type Options struct {
	MetadataFileName string
	InvoicesCSVName  string
	RowsCSVName      string
}

type Result struct {
	InvoicesCSVPath string
	RowsCSVPath     string
	InvoiceCount    int
	RowCount        int
}

func ExportInvoicesFromZIP(zipPath, outputDir string, opts Options) (Result, error) {
	opts = opts.withDefaults()

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output directory: %w", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return Result{}, fmt.Errorf("open ZIP file: %w", err)
	}
	defer reader.Close()

	result := Result{
		InvoicesCSVPath: filepath.Join(outputDir, opts.InvoicesCSVName),
		RowsCSVPath:     filepath.Join(outputDir, opts.RowsCSVName),
	}

	invoiceCount, err := exportMetadataCSV(&reader.Reader, opts.MetadataFileName, result.InvoicesCSVPath)
	if err != nil {
		return Result{}, err
	}
	result.InvoiceCount = invoiceCount

	rowCount, err := exportRowsCSV(&reader.Reader, result.RowsCSVPath)
	if err != nil {
		return Result{}, err
	}
	result.RowCount = rowCount

	return result, nil
}

func (o Options) withDefaults() Options {
	if o.MetadataFileName == "" {
		o.MetadataFileName = DefaultMetadataFileName
	}
	if o.InvoicesCSVName == "" {
		o.InvoicesCSVName = DefaultInvoicesCSVName
	}
	if o.RowsCSVName == "" {
		o.RowsCSVName = DefaultRowsCSVName
	}
	return o
}

func exportMetadataCSV(reader *zip.Reader, metadataFileName, outputPath string) (int, error) {
	metadataFile := findZipFile(reader, metadataFileName)
	if metadataFile == nil {
		return 0, fmt.Errorf("metadata file %q not found in ZIP", metadataFileName)
	}

	in, err := metadataFile.Open()
	if err != nil {
		return 0, fmt.Errorf("open metadata file %q: %w", metadataFile.Name, err)
	}
	defer in.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("create invoice metadata CSV file: %w", err)
	}
	defer out.Close()

	count, err := WriteInvoiceMetadataCSVFromJSON(in, out)
	if err != nil {
		return count, fmt.Errorf("export invoice metadata CSV: %w", err)
	}
	return count, nil
}

func exportRowsCSV(reader *zip.Reader, outputPath string) (int, error) {
	out, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("create invoice rows CSV file: %w", err)
	}
	defer out.Close()

	writer := csv.NewWriter(out)
	if err := WriteInvoiceRowsCSVHeader(writer); err != nil {
		return 0, fmt.Errorf("write invoice rows CSV header: %w", err)
	}

	total := 0
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !isInvoiceXMLFile(file.Name) {
			continue
		}

		in, err := file.Open()
		if err != nil {
			return total, fmt.Errorf("open invoice XML file %q: %w", file.Name, err)
		}

		ksefNumber := strings.TrimSuffix(path.Base(file.Name), path.Ext(file.Name))
		count, err := ParseInvoiceRowsXML(in, ksefNumber, file.Name, func(row InvoiceRow) error {
			return writer.Write(InvoiceRowCSVRecord(row))
		})
		closeErr := in.Close()
		if err != nil {
			return total, fmt.Errorf("parse invoice XML file %q: %w", file.Name, err)
		}
		if closeErr != nil {
			return total, fmt.Errorf("close invoice XML file %q: %w", file.Name, closeErr)
		}
		total += count
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return total, fmt.Errorf("write invoice rows CSV: %w", err)
	}

	return total, nil
}

func findZipFile(reader *zip.Reader, name string) *zip.File {
	for _, file := range reader.File {
		if path.Base(file.Name) == name {
			return file
		}
	}
	return nil
}

func isInvoiceXMLFile(name string) bool {
	base := path.Base(name)
	return strings.EqualFold(path.Ext(base), ".xml") && !strings.HasPrefix(base, "_")
}
