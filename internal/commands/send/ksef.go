package send

import (
	"context"
	"fmt"
	"go-ksef-cli/internal/app"
	"io"
	"os"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/aes"
	"github.com/alapierre/go-ksef-client/ksef/api"
)

func prepareToSend(appContext *app.App, nip string) ([]byte, []byte, string, error) {

	key, err := aes.GenerateRandom256BitsKey()
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate AES key: %w", err)
	}

	iv, err := aes.GenerateRandom16BytesIv()
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate AES IV: %w", err)
	}

	ctx := ksef.ContextWithEnv(context.Background(), nip, appContext.Env)

	session, err := appContext.Client.OpenInteractiveSession(ctx, defaultFormCode(), key, iv)
	if err != nil {
		return nil, nil, "", fmt.Errorf("open interactive session: %w", err)
	}

	return key, iv, string(session.ReferenceNumber), nil
}

func defaultFormCode() api.FormCode {
	return api.FormCode{
		SystemCode:    "FA (3)",
		SchemaVersion: "1-0E",
		Value:         "FA",
	}
}

func sendToKsef(nip, session string, key, iv []byte, filePath string, appContext *app.App) (string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Warnf("error processing file %s %v", filePath, err)
		}
	}(file)

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	ctx := ksef.ContextWithEnv(context.Background(), nip, appContext.Env)
	ref, err := appContext.Client.SendInvoice(ctx, session, api.NewOptBool(false), data, key, iv)
	if err != nil {
		return "", err
	}

	return ref, nil
}
