package main

import (
	"fmt"
	"github.com/alapierre/go-ksef-client/ksef/model"
	"github.com/cheggaaa/pb/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func initCommand() {
	if checkIsEncKeyInitialized() {
		exitWithError("Encryption key is already stored in keystore")
	}
	err := initEncryption()
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot initalize encryption key: %v", err))
	}
	fmt.Println("Encryption key generated and stored")
}

func logoutCommand(sessionToken *string) { // TODO: zmienić wskaźnik na string

	if *sessionToken == "" {
		fmt.Println("Loading stored session token")
		t := loadSessionToken(c.env)
		sessionToken = &t
	}

	terminate, err := c.sessionService.Terminate(*sessionToken)
	handleError(err)
	fmt.Printf("responce: %#v\n", *terminate)
}

func loginCommand(token, identifier, pathToKey *string) { // TODO: zmienić wskaźniki na stringi

	if *token == "" {
		fmt.Println("Loading stored authorisation token")
		t := loadAuthToken(*identifier, c.env)
		//fmt.Println("token: " + t)
		token = &t
	}

	fmt.Println("Trying to login into KSeF")

	sessionToken, err := c.sessionService.LoginByToken(*identifier, model.ONIP, *token, *pathToKey)
	handleError(err)

	if viper.GetBool("printSessionToken") {
		fmt.Printf("session token: %s refernece number: %s\n", sessionToken.SessionToken.Token, sessionToken.ReferenceNumber)
	} else {
		fmt.Printf("session refernece number: %s\n", sessionToken.ReferenceNumber)
	}

	storeSessionToken(sessionToken.SessionToken.Token, c.env)
}

func sendCommand(sessionToken *string, file *string) {

	if *file == "" {
		exitWithError("there is no file name to send")
	}

	fi, err := os.Stat(*file)
	if err != nil {
		exitWithError(fmt.Sprintf("error read file info, %v", err))
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		processFilesInDir(checkTokenExistOrLoad(sessionToken), file)
	case mode.IsRegular():
		_, err := sendToKsef(checkTokenExistOrLoad(sessionToken), *file)
		if err != nil {
			exitWithError("Error sending file " + *file)
		}
	default:
		exitWithError("Unknown file type")
	}
}

func processFilesInDir(sessionToken string, file *string) {
	files, err := os.ReadDir(*file)
	if err != nil {
		exitWithError(fmt.Sprintf("Error sending dir %v", err))
	}

	count := countFiles(files)
	bar := pb.StartNew(count)
	errors := 0

	results := make([]*model.SendInvoiceResponse, 0)

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".xml") {
			filePath := filepath.Join(*file, f.Name())
			bar.Increment()
			status, err := sendToKsef(sessionToken, filePath)
			if err != nil {
				errors++
				results = append(results, &model.SendInvoiceResponse{ProcessingDescription: err.Error()})
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

func sendToKsef(token string, filePath string) (*model.SendInvoiceResponse, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	invoice, err := c.invoiceService.SendInvoice(data, token)
	if err != nil {
		return nil, err
	}
	return invoice, nil
}

func statusCommand(token *string) error {

	status, err := c.sessionService.Status(100, 0, checkTokenExistOrLoad(token))
	if err != nil {
		logrus.Errorf("Failed sening file: %s", err)
		return err
	}
	printSessionStatus(status)
	return nil
}

func checkTokenExistOrLoad(token *string) string {

	if *token == "" {
		fmt.Println("Loading stored session token")
		t := loadSessionToken(c.env)
		return t
	}
	return *token
}
