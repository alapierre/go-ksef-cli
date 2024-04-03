package main

import (
	"github.com/alapierre/go-ksef-client/ksef/model"
	"github.com/jedib0t/go-pretty/v6/table"
	"os"
)

func printSessionStatus(status *model.SessionStatusResponse) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "KSeF Reference Number", "Invoice Number", "Processing Code", "Acquisition Timestamp", "Processing Description"})

	for i, s := range status.InvoiceStatusList {
		t.AppendRows([]table.Row{
			{i, s.KsefReferenceNumber, s.InvoiceNumber, s.ProcessingCode, s.AcquisitionTimestamp, s.ProcessingDescription},
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}

func printInvoiceSendStatus(invoices []*model.SendInvoiceResponse) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Reference Number", "Timestamp", "Processing Code", "Processing Description"})

	for i, inv := range invoices {
		t.AppendRows([]table.Row{
			{i, inv.ElementReferenceNumber, inv.Timestamp, inv.ProcessingCode, inv.ProcessingDescription},
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}
