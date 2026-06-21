package send

import (
	"context"
	"fmt"
	"go-ksef-cli/internal/app"
	"go-ksef-cli/internal/config"
	"net/http"
	"os"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/alapierre/go-ksef-client/ksef/batch"
	"github.com/cheggaaa/pb/v3"
)

const defaultBatchMaxPartSize int64 = 100 * 1024 * 1024

type BatchCmd struct {
	Paths       []string `arg:"" name:"path" help:"Files or directories to process." type:"path"`
	Recursive   bool     `short:"r" help:"Process directories recursively."`
	Token       string   `short:"t" env:"KSEF_TOKEN" optional:"" help:"KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"`
	Identifier  string   `short:"i" required:"" help:"context identifier (NIP)"`
	OutputDir   string   `help:"Directory for temporary batch files. Defaults to the system temp directory." type:"path"`
	MaxPartSize int64    `default:"104857600" help:"Maximum plain ZIP part size in bytes before encryption."`
	KeepFiles   bool     `help:"Keep generated ZIP and encrypted part files after sending."`
}

func (c *BatchCmd) Run(cfg *config.Config) error {
	xmlPaths, err := collectXMLPaths(c.Paths, c.Recursive)
	if err != nil {
		return err
	}

	if c.MaxPartSize <= 0 {
		c.MaxPartSize = defaultBatchMaxPartSize
	}

	appCtx, err := app.New(c.Token, cfg.Env, c.Identifier)
	if err != nil {
		return err
	}
	ctx := ksef.ContextWithEnv(context.Background(), c.Identifier, appCtx.Env)

	result, err := batch.BuildBatchFromSource(batch.BatchConfig{
		OutputDir:       c.OutputDir,
		MaxPartSize:     c.MaxPartSize,
		TempFilePattern: "ksef-batch-*.zip",
		CleanupPlainZip: !c.KeepFiles,
	}, batch.NewFileInvoiceSource(xmlPaths))
	if err != nil {
		return err
	}
	if !c.KeepFiles {
		defer cleanupBatchParts(result)
	}

	openResp, err := appCtx.Client.OpenBatchSession(ctx, defaultFormCode(), result.AESKey, result.IV, api.OptBool{}, batchFileInfo(result))
	if err != nil {
		return err
	}

	sessionID := string(openResp.ReferenceNumber)
	fmt.Printf("Batch session ID: %s\n", sessionID)
	fmt.Printf("Invoices in batch: %d\n", len(result.InvoiceHashes))
	fmt.Printf("Batch parts: %d\n", len(result.Parts))

	if err := uploadBatchParts(ctx, appCtx, result, openResp.PartUploadRequests); err != nil {
		return err
	}

	closedSessionID, err := appCtx.Client.CloseBatchSession(ctx, sessionID)
	if err != nil {
		return err
	}

	fmt.Printf("Batch session closed: %s\n", closedSessionID)
	return nil
}

func batchFileInfo(result *batch.BatchResult) api.BatchFileInfo {
	fileParts := make([]api.BatchFilePartInfo, 0, len(result.Parts))
	for _, part := range result.Parts {
		fileParts = append(fileParts, api.BatchFilePartInfo{
			OrdinalNumber: int32(part.Index + 1),
			FileSize:      part.CipherSize,
			FileHash:      api.Sha256HashBase64(part.CipherSHA256),
		})
	}

	return api.BatchFileInfo{
		FileSize:  result.ZipSize,
		FileHash:  api.Sha256HashBase64(result.ZipSHA256),
		FileParts: fileParts,
	}
}

func uploadBatchParts(ctx context.Context, appCtx *app.App, result *batch.BatchResult, requests []api.PartUploadRequest) error {
	if len(requests) != len(result.Parts) {
		return fmt.Errorf("KSeF requested %d batch parts, but built %d", len(requests), len(result.Parts))
	}

	bar := pb.StartNew(len(requests))
	defer bar.Finish()

	for _, request := range requests {
		ordinal := int(request.OrdinalNumber)
		if ordinal <= 0 || ordinal > len(result.Parts) {
			return fmt.Errorf("KSeF returned invalid batch part ordinal number %d", ordinal)
		}

		part := result.Parts[ordinal-1]
		data, err := os.ReadFile(part.CipherPath)
		if err != nil {
			return fmt.Errorf("read encrypted batch part %d from %s: %w", ordinal, part.CipherPath, err)
		}

		response, err := appCtx.Client.SendBatchPart(ctx, data, request)
		if err != nil {
			statusCode := 0
			body := ""
			if response != nil {
				statusCode = response.StatusCode
				body = response.Message
			}
			return fmt.Errorf("send batch part %d failed: %w (status=%d, body=%s)", ordinal, err, statusCode, body)
		}
		if response.StatusCode != http.StatusCreated {
			return fmt.Errorf("send batch part %d returned HTTP %d: %s", ordinal, response.StatusCode, response.Message)
		}

		bar.Increment()
	}

	return nil
}

func cleanupBatchParts(result *batch.BatchResult) {
	for _, part := range result.Parts {
		if err := os.Remove(part.CipherPath); err != nil && !os.IsNotExist(err) {
			logger.Warnf("error removing batch part %s: %v", part.CipherPath, err)
		}
	}
}
