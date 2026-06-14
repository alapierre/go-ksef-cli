package invoicereport

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/alapierre/go-ksef-client/ksef/api"
)

func WriteInvoiceMetadataCSV(path string, r *api.QueryInvoicesMetadataResponse) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create invoice metadata CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(InvoiceMetadataCSVRecords(r)); err != nil {
		return fmt.Errorf("write invoice metadata CSV file: %w", err)
	}

	return nil
}

func WriteInvoiceMetadataCSVFromJSON(in io.Reader, out io.Writer) (int, error) {
	writer := csv.NewWriter(out)
	if err := writer.Write(InvoiceMetadataCSVHeader()); err != nil {
		return 0, fmt.Errorf("write invoice metadata CSV header: %w", err)
	}

	count, err := decodeInvoiceMetadataJSON(in, func(inv api.InvoiceMetadata) error {
		return writer.Write(InvoiceMetadataCSVRecord(inv))
	})
	if err != nil {
		return count, err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return count, fmt.Errorf("write invoice metadata CSV: %w", err)
	}

	return count, nil
}

func InvoiceMetadataCSVRecords(r *api.QueryInvoicesMetadataResponse) [][]string {
	records := [][]string{InvoiceMetadataCSVHeader()}
	for _, inv := range r.Invoices {
		records = append(records, InvoiceMetadataCSVRecord(inv))
	}
	return records
}

func InvoiceMetadataCSVHeader() []string {
	return []string{
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
	}
}

func InvoiceMetadataCSVRecord(inv api.InvoiceMetadata) []string {
	return []string{
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
	}
}

func decodeInvoiceMetadataJSON(in io.Reader, handle func(api.InvoiceMetadata) error) (int, error) {
	decoder := json.NewDecoder(in)

	token, err := decoder.Token()
	if err != nil {
		return 0, fmt.Errorf("decode invoice metadata JSON: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return 0, fmt.Errorf("decode invoice metadata JSON: expected object")
	}

	count := 0
	foundInvoices := false
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return count, fmt.Errorf("decode invoice metadata JSON field: %w", err)
		}
		field, ok := token.(string)
		if !ok {
			return count, fmt.Errorf("decode invoice metadata JSON: expected field name")
		}

		if field != "invoices" {
			var ignored json.RawMessage
			if err := decoder.Decode(&ignored); err != nil {
				return count, fmt.Errorf("decode invoice metadata JSON field %q: %w", field, err)
			}
			continue
		}
		foundInvoices = true

		token, err = decoder.Token()
		if err != nil {
			return count, fmt.Errorf("decode invoices array: %w", err)
		}
		if delimiter, ok := token.(json.Delim); !ok || delimiter != '[' {
			return count, fmt.Errorf("decode invoices array: expected array")
		}
		for decoder.More() {
			var inv api.InvoiceMetadata
			if err := decoder.Decode(&inv); err != nil {
				return count, fmt.Errorf("decode invoice metadata item %d: %w", count+1, err)
			}
			if err := handle(inv); err != nil {
				return count, fmt.Errorf("handle invoice metadata item %d: %w", count+1, err)
			}
			count++
		}
		token, err = decoder.Token()
		if err != nil {
			return count, fmt.Errorf("decode invoices array end: %w", err)
		}
		if delimiter, ok := token.(json.Delim); !ok || delimiter != ']' {
			return count, fmt.Errorf("decode invoices array: expected array end")
		}
	}

	token, err = decoder.Token()
	if err != nil {
		return count, fmt.Errorf("decode invoice metadata JSON end: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return count, fmt.Errorf("decode invoice metadata JSON: expected object end")
	}
	if !foundInvoices {
		return count, fmt.Errorf("decode invoice metadata JSON: missing invoices field")
	}

	return count, nil
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
