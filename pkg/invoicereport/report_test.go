package invoicereport

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseInvoiceRowsXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<Faktura xmlns="http://crd.gov.pl/wzor/2025/06/25/13775/">
	<Fa>
		<FaWiersz>
			<NrWierszaFa>1</NrWierszaFa>
			<P_7>Sprzedaz towarow 23%</P_7>
			<P_8A>szt.</P_8A>
			<P_8B>2.323</P_8B>
			<P_9A>234.24</P_9A>
			<P_11>544.14</P_11>
			<P_12>zw</P_12>
		</FaWiersz>
		<FaWiersz>
			<NrWierszaFa>2</NrWierszaFa>
			<P_7>GTU_1</P_7>
			<P_8A>-</P_8A>
			<P_8B>2.561</P_8B>
			<P_9A>1350.00</P_9A>
			<P_11>3457.35</P_11>
			<P_12>zw</P_12>
		</FaWiersz>
	</Fa>
</Faktura>`

	var rows []InvoiceRow
	count, err := ParseInvoiceRowsXML(strings.NewReader(xml), "123", "123.xml", func(row InvoiceRow) error {
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseInvoiceRowsXML() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if rows[0].KsefNumber != "123" {
		t.Fatalf("KsefNumber = %q, want 123", rows[0].KsefNumber)
	}
	if got, want := rows[0].Fields["P_7"], "Sprzedaz towarow 23%"; got != want {
		t.Fatalf("P_7 = %q, want %q", got, want)
	}
	if got, want := rows[1].Fields["P_11"], "3457.35"; got != want {
		t.Fatalf("P_11 = %q, want %q", got, want)
	}
}

func TestExportInvoicesFromZIP(t *testing.T) {
	outputDir := t.TempDir()
	result, err := ExportInvoicesFromZIP(filepath.Join("..", "..", "_test", "export.zip"), outputDir, Options{})
	if err != nil {
		t.Fatalf("ExportInvoicesFromZIP() error = %v", err)
	}

	if result.InvoiceCount != 5 {
		t.Fatalf("InvoiceCount = %d, want 5", result.InvoiceCount)
	}
	if result.RowCount != 10 {
		t.Fatalf("RowCount = %d, want 10", result.RowCount)
	}

	invoiceRecords := readCSV(t, result.InvoicesCSVPath)
	if got, want := len(invoiceRecords), 6; got != want {
		t.Fatalf("invoice CSV record count = %d, want %d", got, want)
	}
	assertRecordEqual(t, invoiceRecords[0], InvoiceMetadataCSVHeader())
	if got, want := invoiceRecords[1][0], "5080092141-20260611-6E8EFB400000-DB"; got != want {
		t.Fatalf("invoice ksef_number = %q, want %q", got, want)
	}

	rowRecords := readCSV(t, result.RowsCSVPath)
	if got, want := len(rowRecords), 11; got != want {
		t.Fatalf("row CSV record count = %d, want %d", got, want)
	}
	assertRecordEqual(t, rowRecords[0], InvoiceRowsCSVHeader())
	if got, want := rowRecords[1][0], "5080092141-20260611-6E8EFB400000-DB"; got != want {
		t.Fatalf("row ksef_number = %q, want %q", got, want)
	}
	if got, want := rowRecords[1][5], "Sprzedaż towarów 23%"; got != want {
		t.Fatalf("row P_7 = %q, want %q", got, want)
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open CSV %s: %v", path, err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read CSV %s: %v", path, err)
	}
	return records
}

func assertRecordEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("record length = %d, want %d\ngot: %#v\nwant: %#v", len(got), len(want), got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("record[%d] = %q, want %q\ngot: %#v\nwant: %#v", i, got[i], want[i], got, want)
		}
	}
}
