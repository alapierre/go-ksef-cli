package send

import (
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "command send")

type Cmd struct {
	Paths      []string `arg:"" name:"path" help:"Files or directories to process." type:"path"`
	Recursive  bool     `short:"r" help:"Process directories recursively."`
	Token      string   `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier string   `short:"i" required:"" help:"context identifier (NIP)"`
}

func (c *Cmd) Run(cfg *config.Config) error {
	xmlPaths, err := collectXMLPaths(c.Paths, c.Recursive)
	if err != nil {
		return err
	}

	token, err := app.ResolveAuthToken(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}

	appCtx := app.New(token, cfg.Env)
	key, iv, session := prepareToSend(appCtx, c.Identifier)

	fmt.Printf("Interactive session ID: %s\n", session)

	errors := 0
	results := make([]string, 0)

	bar := pb.StartNew(len(xmlPaths))

	for _, path := range xmlPaths {
		if status, err := processFile(appCtx, key, iv, c.Identifier, session, path); err != nil {
			errors++
			results = append(results, err.Error())
		} else {
			results = append(results, status)
		}
		bar.Increment()
	}

	bar.Finish()

	if errors > 0 {
		fmt.Printf("Error sending %d files of total: %d\n", errors, len(xmlPaths))
	} else {
		fmt.Printf("Successfully sent %d files\n", len(xmlPaths))
	}

	printInvoiceSendStatus(results)

	return nil
}

func collectXMLPaths(paths []string, recursive bool) ([]string, error) {
	xmlPaths := make([]string, 0)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}

		if info.IsDir() {
			dirPaths, err := collectXMLPathsFromDir(path, recursive)
			if err != nil {
				return nil, err
			}
			xmlPaths = append(xmlPaths, dirPaths...)
			continue
		}

		if isXMLPath(path) {
			xmlPaths = append(xmlPaths, path)
		}
	}

	return xmlPaths, nil
}

func collectXMLPathsFromDir(dir string, recursive bool) ([]string, error) {
	xmlPaths := make([]string, 0)

	if recursive {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if isXMLPath(path) {
				xmlPaths = append(xmlPaths, path)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		return xmlPaths, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if isXMLPath(path) {
			xmlPaths = append(xmlPaths, path)
		}
	}

	return xmlPaths, nil
}

func isXMLPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".xml")
}

func processFile(appContext *app.App, key, iv []byte, nip, session string, path string) (string, error) {
	status, err := sendToKsef(nip, session, key, iv, path, appContext)
	if err == nil {
		logger.Infof("File %s sent with status %s", path, status)
	} else {
		logger.Warnf("Error sending file %s %v", path, err)
	}
	return status, err
}

func printInvoiceSendStatus(invoices []string) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Reference Number"})

	for i, inv := range invoices {
		t.AppendRows([]table.Row{
			{i + 1, inv},
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}
