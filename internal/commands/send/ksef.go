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

func prepareToSend(appContext *app.App, nip string) ([]byte, []byte, string) {

	form := api.FormCode{
		SystemCode:    "FA (3)",
		SchemaVersion: "1-0E",
		Value:         "FA",
	}

	key, err := aes.GenerateRandom256BitsKey()
	iv, err := aes.GenerateRandom16BytesIv()

	ctx := ksef.ContextWithEnv(context.Background(), nip, appContext.Env)

	session, err := appContext.Client.OpenInteractiveSession(ctx, form, key, iv)
	if err != nil {
		fmt.Printf("error opening session %v\n", err)
		logger.Fatal(err)
	}

	return key, iv, string(session.ReferenceNumber)
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
