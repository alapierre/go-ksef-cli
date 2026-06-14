package invoicereport

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type InvoiceRow struct {
	KsefNumber string
	SourceFile string
	Fields     map[string]string
}

var invoiceRowFields = []string{
	"NrWierszaFa",
	"UU_ID",
	"P_6A",
	"P_7",
	"Indeks",
	"GTIN",
	"PKWiU",
	"CN",
	"PKOB",
	"P_8A",
	"P_8B",
	"P_9A",
	"P_9B",
	"P_10",
	"P_11",
	"P_11A",
	"P_11Vat",
	"P_12",
	"P_12_XII",
	"P_12_Zal_15",
	"KwotaAkcyzy",
	"GTU",
	"Procedura",
	"KursWaluty",
	"StanPrzed",
}

func WriteInvoiceRowsCSVHeader(writer *csv.Writer) error {
	return writer.Write(InvoiceRowsCSVHeader())
}

func InvoiceRowsCSVHeader() []string {
	header := []string{"ksef_number", "source_file"}
	return append(header, invoiceRowFields...)
}

func InvoiceRowCSVRecord(row InvoiceRow) []string {
	record := []string{row.KsefNumber, row.SourceFile}
	for _, field := range invoiceRowFields {
		record = append(record, row.Fields[field])
	}
	return record
}

func ParseInvoiceRowsXML(in io.Reader, ksefNumber, sourceFile string, handle func(InvoiceRow) error) (int, error) {
	decoder := xml.NewDecoder(in)
	count := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return count, fmt.Errorf("decode XML token: %w", err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "FaWiersz" {
			continue
		}

		row, err := decodeInvoiceRow(decoder, start, ksefNumber, sourceFile)
		if err != nil {
			return count, fmt.Errorf("decode FaWiersz %d: %w", count+1, err)
		}
		if err := handle(row); err != nil {
			return count, fmt.Errorf("handle FaWiersz %d: %w", count+1, err)
		}
		count++
	}
}

func decodeInvoiceRow(decoder *xml.Decoder, start xml.StartElement, ksefNumber, sourceFile string) (InvoiceRow, error) {
	row := InvoiceRow{
		KsefNumber: ksefNumber,
		SourceFile: sourceFile,
		Fields:     make(map[string]string, len(invoiceRowFields)),
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			return row, err
		}

		switch token := token.(type) {
		case xml.StartElement:
			var value string
			if err := decoder.DecodeElement(&value, &token); err != nil {
				return row, err
			}
			row.Fields[token.Name.Local] = strings.TrimSpace(value)
		case xml.EndElement:
			if token.Name.Local == start.Name.Local {
				return row, nil
			}
		}
	}
}
