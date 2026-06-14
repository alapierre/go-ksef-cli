package cli

import (
	"go-ksef-cli/internal/commands/initcmd"
	"go-ksef-cli/internal/commands/invoice"
	"go-ksef-cli/internal/commands/login"
	"go-ksef-cli/internal/commands/printcmd"
	"go-ksef-cli/internal/commands/query"
	"go-ksef-cli/internal/commands/report"
	"go-ksef-cli/internal/commands/send"
	"go-ksef-cli/internal/commands/session"
	"go-ksef-cli/internal/commands/store"
	"go-ksef-cli/internal/commands/version"
	"go-ksef-cli/internal/config"
)

type CLI struct {
	Config config.Config `embed:""`

	Init    initcmd.Cmd  `cmd:"init" help:"Initialize encryption key and save it in keystore selected in configuration'"`
	Login   login.Cmd    `cmd:"login" help:"Login into KSeF using provided authorisation token and store encrypted session tokens'"`
	Store   store.Cmd    `cmd:"store" help:"Encrypt and store KSeF authorisation token'"`
	Print   printcmd.Cmd `cmd:"print" help:"Print stored KSeF session tokens'"`
	Send    send.Cmd     `cmd:"send" help:"Send XML Invoice files to KSeF'"`
	Query   query.Cmd    `cmd:"query" help:"Query invoices form KSeF"`
	Report  report.Cmd   `cmd:"report" help:"Create CSV reports from exported data"`
	Session session.Cmd  `cmd:"session" help:"Manage KSeF session"`
	Invoice invoice.Cmd  `cmd:"invoice" help:"Manage invoices"`
	Version version.Cmd  `cmd:"version" help:"Print CLI version"`
}
