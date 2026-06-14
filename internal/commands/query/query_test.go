package query

import (
	"testing"
	"time"

	"github.com/alapierre/go-ksef-client/ksef/api"

	"go-ksef-cli/pkg/invoicereport"
)

func TestInvoiceMetadataCSVRecords(t *testing.T) {
	records := invoicereport.InvoiceMetadataCSVRecords(&api.QueryInvoicesMetadataResponse{
		Invoices: []api.InvoiceMetadata{
			{
				KsefNumber:           api.KsefNumber("123456789012345678901234567890123456"),
				InvoiceNumber:        "FV/1/2026",
				IssueDate:            time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				InvoicingDate:        time.Date(2026, 6, 2, 10, 11, 12, 0, time.UTC),
				AcquisitionDate:      time.Date(2026, 6, 2, 10, 12, 13, 0, time.UTC),
				PermanentStorageDate: time.Date(2026, 6, 2, 10, 13, 14, 0, time.UTC),
				Seller: api.InvoiceMetadataSeller{
					Nip:  api.Nip("1111111111"),
					Name: api.NewOptNilString("Seller SA"),
				},
				Buyer: api.InvoiceMetadataBuyer{
					Identifier: api.InvoiceMetadataBuyerIdentifier{
						Type:  api.BuyerIdentifierTypeNip,
						Value: api.NewOptNilString("2222222222"),
					},
					Name: api.NewOptNilString("Buyer Sp. z o.o."),
				},
				NetAmount:       100.5,
				GrossAmount:     123.62,
				VatAmount:       23.12,
				Currency:        "PLN",
				InvoicingMode:   api.InvoicingModeOnline,
				InvoiceType:     api.InvoiceTypeVat,
				FormCode:        api.FormCode{SystemCode: "FA (3)", SchemaVersion: "1-0E", Value: "FA"},
				IsSelfInvoicing: true,
				HasAttachment:   true,
				InvoiceHash:     api.Sha256HashBase64{1, 2, 3},
				HashOfCorrectedInvoice: api.NewOptNilSha256HashBase64(
					api.Sha256HashBase64{4, 5, 6},
				),
				ThirdSubjects: api.NewOptNilInvoiceMetadataThirdSubjectArray([]api.InvoiceMetadataThirdSubject{
					{
						Identifier: api.InvoiceMetadataThirdSubjectIdentifier{
							Type:  api.ThirdSubjectIdentifierTypeVatUe,
							Value: api.NewOptNilString("PL123"),
						},
						Name: api.NewOptNilString("Third Subject"),
						Role: 4,
					},
				}),
				AuthorizedSubject: api.NewOptNilInvoiceMetadataAuthorizedSubject(api.InvoiceMetadataAuthorizedSubject{
					Nip:  api.Nip("3333333333"),
					Name: api.NewOptNilString("Authorized Subject"),
					Role: 3,
				}),
			},
		},
	})

	if got, want := len(records), 2; got != want {
		t.Fatalf("records length = %d, want %d", got, want)
	}

	wantHeader := []string{
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

	assertRecordEqual(t, records[0], wantHeader)
	assertRecordEqual(t, records[1], []string{
		"123456789012345678901234567890123456",
		"FV/1/2026",
		"2026-06-01",
		"2026-06-02T10:11:12Z",
		"2026-06-02T10:12:13Z",
		"2026-06-02T10:13:14Z",
		"1111111111",
		"Seller SA",
		"Nip",
		"2222222222",
		"Buyer Sp. z o.o.",
		"100.5",
		"123.62",
		"23.12",
		"PLN",
		"Online",
		"Vat",
		"FA (3)",
		"1-0E",
		"FA",
		"true",
		"true",
		"AQID",
		"BAUG",
		`[{"role":4,"identifier_type":"VatUe","identifier_value":"PL123","name":"Third Subject"}]`,
		"3333333333",
		"Authorized Subject",
		"3",
	})
}

func TestFiltersOmitDateToWhenNotProvided(t *testing.T) {
	cmd := Cmd{
		DateFrom:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		SubjectType: "Subject1",
		DateType:    "PermanentStorage",
		FormType:    "FA",
	}

	filters := cmd.filers()

	if filters.DateRange.To.IsSet() {
		t.Fatalf("DateRange.To should not be set when DateTo is zero")
	}
}

func TestFiltersSetDateToWhenProvided(t *testing.T) {
	dateTo := time.Date(2026, 6, 14, 11, 0, 0, 0, time.UTC)
	cmd := Cmd{
		DateFrom:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		DateTo:      dateTo,
		SubjectType: "Subject1",
		DateType:    "PermanentStorage",
		FormType:    "FA",
	}

	filters := cmd.filers()
	got, ok := filters.DateRange.To.Get()
	if !ok {
		t.Fatalf("DateRange.To should be set when DateTo is provided")
	}
	if !got.Equal(dateTo) {
		t.Fatalf("DateRange.To = %s, want %s", got, dateTo)
	}
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
