package main

import (
	"fmt"
	"go-ksef-cli/internal/cli"
	"go-ksef-cli/internal/config"
	"os"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "KSeF CLI")

func main() {

	file, err := os.OpenFile(
		"ksef-cli.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		logger.Fatalf("cannot open log file: %v", err)
	}
	defer file.Close()

	logrus.SetOutput(file)

	logger.Info("Application started")

	var appCli cli.CLI

	ctx := kong.Parse(&appCli,
		kong.Name(config.AppName),
		kong.Description("KSeF command line interface"),
		kong.Bind(&appCli.Config),
	)

	ctx.FatalIfErrorf(ctx.Run())

	//logoutCmd := parser.NewCommand("logout", "logout from KSeF by close interactive session")
	//sessionToken := logoutCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token"})
	//
	//sendInvoiceCmd := parser.NewCommand("send", "send invoice into KSeF")
	//sendToken := sendInvoiceCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token, if not provided it will be loaded (you should login first"})
	//fileToSend := sendInvoiceCmd.StringPositional(&argparse.Options{Required: true, Help: "XML invoice file or directory"})
}

func exitWithError(message string) {
	fmt.Println(message)
	os.Exit(1)
}
