package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/alapierre/go-ksef-client/ksef/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

type context struct {
	pathToKey         string
	env               string
	sessionService    api.SessionService
	loadEncryptionKey func() []byte
	initEncryption    func() error
}

var c context

func main() {

	parser := argparse.NewParser("ksef-cli", "KSeF Command line interface")

	loginCmd := parser.NewCommand("login", "login into KSeF using provided authorisation token")
	token := loginCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"})
	identifier := loginCmd.String("i", "identifier", &argparse.Options{Required: true, Help: "Organization identifier (NIP)"})

	logoutCmd := parser.NewCommand("logout", "logout from KSeF by close interactive session")
	sessionToken := logoutCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token"})

	initCmd := parser.NewCommand("init", "initialize encryption key and save it in keystore selected in configuration")

	storeAuthTokenCmd := parser.NewCommand("store", "encrypt and store authorisation token in keystore selected in configuration")
	tokenToStore := storeAuthTokenCmd.String("t", "token", &argparse.Options{Required: true, Help: "KSeF authorisation token"})
	identifierToStore := storeAuthTokenCmd.String("i", "identifier", &argparse.Options{Required: true, Help: "Organization identifier (NIP)"})

	config()

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	client := api.New(stringToEnvName(c.env))
	c.sessionService = api.NewSessionService(client)

	if loginCmd.Happened() {
		loginCommand(token, identifier, &c.pathToKey)
	} else if logoutCmd.Happened() {
		logoutCommand(sessionToken)
	} else if initCmd.Happened() {
		initCommand()
	} else if storeAuthTokenCmd.Happened() {
		storeAuthToken(*tokenToStore, *identifierToStore, c.env)
		fmt.Printf("Token for identifier %s for envitnoment: %s stored successfully\n", *identifier, c.env)
	}

}

func handleError(err error) {
	if err != nil {
		re, ok := err.(*api.RequestError)
		if ok {
			log.Errorf("request error %d responce body %s", re.StatusCode, re.Body)
			os.Exit(1)
		}
		panic(err)
	}
}

func config() {
	viper.SetConfigName("config.env")
	viper.SetConfigType("env")
	viper.AddConfigPath("$HOME/.go-ksef-cli")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("KSEF_")
	viper.AutomaticEnv()

	viper.SetDefault("env", "test")
	viper.SetDefault("mfKeys", "keys")
	viper.SetDefault("keystore", "desktop")
	viper.SetDefault("printSessionToken", "true")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("WARNING: Can't load %s\n", err)
	}

	keysPath := viper.GetString("mfKeys")
	ksefEnv := viper.GetString("env")
	c.pathToKey = fmt.Sprintf("%s/%s/publicKey.pem", keysPath, ksefEnv)
	c.env = ksefEnv

}

func stringToEnvName(env string) api.Environment {
	if env == "demo" {
		return api.Demo
	} else if env == "prod" {
		return api.Prod
	} else {
		return api.Test
	}
}

func exitWithError(message string) {
	fmt.Println(message)
	os.Exit(1)
}
