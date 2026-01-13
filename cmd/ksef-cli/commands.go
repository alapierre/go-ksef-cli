package main

import (
	"context"
	"fmt"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/aes"
	"github.com/alapierre/go-ksef-client/ksef/api"

	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/viper"
)

func initCommand(force bool) {
	if force || checkIsEncKeyInitialized() {
		exitWithError("Encryption key is already stored in keystore")
	}
	err := initEncryption()
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot initialize encryption key: %v", err))
	}
	fmt.Println("Encryption key generated and stored")
}

func logoutCommand(sessionToken *string) { // TODO: zmienić wskaźnik na string
	fmt.Printf("not implemented yet\n")
}

func loginCommand(ctx context.Context, token string, appContext *app) {

	fmt.Println("Trying to login into KSeF")

	ksefToken, err := ksef.WithKsefToken(ctx, appContext.authFacade, appContext.encryptor, token)
	if err != nil {
		return
	}

	if viper.GetBool("printSessionToken") {
		printTokens(ksefToken)
	} else {
		fmt.Printf("login successful")
	}

	nip, ok := ksef.NipFromContext(ctx)
	if !ok {
		logger.Fatal("No NIP in context")
	}

	e, ok := ksef.EnvFromContext(ctx)
	if !ok {
		logger.Fatal("No env in context")
	}

	storeSessionToken(ksefToken, e.Name(), nip)
}

func sendCommand(file *string, appContext *app) {

	if *file == "" {
		exitWithError("there is no file name to send")
	}

	fi, err := os.Stat(*file)
	if err != nil {
		exitWithError(fmt.Sprintf("error read file info, %v", err))
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		processFilesInDir(file, appContext)
	case mode.IsRegular():
		key, iv, session := prepareToSend(appContext)
		_, err := sendToKsef(session, key, iv, *file, appContext)
		if err != nil {
			exitWithError("Error sending file " + *file)
		}
	default:
		exitWithError("Unknown file type")
	}
}

func prepareToSend(appContext *app) ([]byte, []byte, string) {

	form := api.FormCode{
		SystemCode:    "FA (3)",
		SchemaVersion: "1-0E",
		Value:         "FA",
	}

	key, err := aes.GenerateRandom256BitsKey()
	iv, err := aes.GenerateRandom16BytesIv()
	encryptedKey, err := appContext.encryptor.EncryptSymmetricKey(context.Background(), key)

	enc := api.EncryptionInfo{
		EncryptedSymmetricKey: encryptedKey,
		InitializationVector:  iv,
	}

	session, err := appContext.client.OpenInteractiveSession(context.Background(), form, enc)
	if err != nil {
		logger.Fatal(err)
	}

	return key, iv, string(session.ReferenceNumber)
}

func processFilesInDir(file *string, appContext *app) {
	files, err := os.ReadDir(*file)
	if err != nil {
		exitWithError(fmt.Sprintf("Error sending dir %v", err))
	}

	count := countFiles(files)
	bar := pb.StartNew(count)
	errors := 0

	ctx := context.Background()
	form := api.FormCode{
		SystemCode:    "FA (3)",
		SchemaVersion: "1-0E",
		Value:         "FA",
	}

	key, err := aes.GenerateRandom256BitsKey()
	iv, err := aes.GenerateRandom16BytesIv()
	encryptedKey, err := appContext.encryptor.EncryptSymmetricKey(ctx, key)

	enc := api.EncryptionInfo{
		EncryptedSymmetricKey: encryptedKey,
		InitializationVector:  iv,
	}

	session, err := appContext.client.OpenInteractiveSession(ctx, form, enc)
	if err != nil {
		logger.Fatal(err)
	}

	results := make([]string, 0)

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".xml") {
			filePath := filepath.Join(*file, f.Name())
			bar.Increment()

			status, err := sendToKsef(string(session.ReferenceNumber), key, iv, filePath, appContext)
			if err != nil {
				errors++
				results = append(results, err.Error())
			} else {
				results = append(results, status)
			}
		}
	}
	bar.Finish()
	if errors > 0 {
		fmt.Printf("Error sending %d files of total: %d\n", errors, count)
	} else {
		fmt.Printf("Successfully sent %d files\n", count)
	}

	printInvoiceSendStatus(results)
}

func countFiles(files []os.DirEntry) int {
	var count int
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".xml") {
			count++
		}
	}
	return count
}

func sendToKsef(session string, key, iv []byte, filePath string, appContext *app) (string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	ref, err := appContext.client.SendInvoice(context.Background(), session, api.NewOptBool(false), data, key, iv)
	if err != nil {
		return "", err
	}

	return ref, nil
}

func statusCommand(ctx context.Context) error {

	nip, ok := ksef.NipFromContext(ctx)
	if !ok {
		logger.Fatal("No NIP in context")
	}

	e, ok := ksef.EnvFromContext(ctx)
	if !ok {
		logger.Fatal("No env in context")
	}

	t := loadSessionToken(e.Name(), nip)
	printTokens(t)

	return nil
}
