package main

import (
	"fmt"
	"github.com/alapierre/go-ksef-client/ksef/model"
	"github.com/spf13/viper"
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
		fmt.Println("token: " + t)
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
