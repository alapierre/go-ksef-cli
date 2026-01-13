package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/akamensky/argparse"
	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type app struct {
	pathToKey     string
	env           string
	client        *ksef.Client
	tokenProvider *ksef.TokenProvider
	encryptor     *ksef.EncryptionService
	authFacade    *ksef.AuthFacade
}

var c app
var logger = logrus.WithField("component", "KSeF CLI")

func main() {

	parser := argparse.NewParser("ksef-cli", "KSeF Command line interface")

	loginCmd := parser.NewCommand("login", "login into KSeF using provided authorisation token")
	identifier := parser.String("i", "identifier", &argparse.Options{Required: true, Help: "Organization identifier (NIP)"})

	token := loginCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF authorisation token, if not provided it will be loaded from keystore (it should be stored first)"})

	//logoutCmd := parser.NewCommand("logout", "logout from KSeF by close interactive session")
	//sessionToken := logoutCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token"})
	//
	//sendInvoiceCmd := parser.NewCommand("send", "send invoice into KSeF")
	//sendToken := sendInvoiceCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token, if not provided it will be loaded (you should login first"})
	//fileToSend := sendInvoiceCmd.StringPositional(&argparse.Options{Required: true, Help: "XML invoice file or directory"})

	initCmd := parser.NewCommand("init", "initialize encryption key and save it in keystore selected in configuration")

	//storeAuthTokenCmd := parser.NewCommand("store", "encrypt and store authorisation token in keystore selected in configuration")
	//tokenToStore := storeAuthTokenCmd.String("t", "token", &argparse.Options{Required: true, Help: "KSeF authorisation token"})
	//identifierToStore := storeAuthTokenCmd.String("i", "identifier", &argparse.Options{Required: true, Help: "Organization identifier (NIP)"})

	statusCmd := parser.NewCommand("status", "print KSeF session status")
	//statusToken := statusCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token, if not provided it will be loaded (you should login first"})

	config()

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	var env ksef.Environment
	err = env.UnmarshalText([]byte(c.env))
	c.authFacade, err = ksef.NewAuthFacade(env, httpClient)

	if err != nil {
		logger.Fatal(err)
	}

	c.encryptor, err = ksef.NewEncryptionService(env, httpClient)
	if err != nil {
		logger.Fatal(err)
	}

	c.tokenProvider = ksef.NewTokenProvider(c.authFacade, func(ctx context.Context) (*api.AuthenticationTokensResponse, error) {
		return ksef.WithKsefToken(ctx, c.authFacade, c.encryptor, *token)
	})

	c.client, err = ksef.NewClient(env, httpClient, c.tokenProvider)

	ctx := ksef.ContextWithEnv(context.Background(), *identifier, env)

	if loginCmd.Happened() {
		loginCommand(ctx, *token)
		//} else if logoutCmd.Happened() {
		//	logoutCommand(sessionToken)
		//} else if sendInvoiceCmd.Happened() {
		//	sendCommand(*sendToken, fileToSend)
	} else if statusCmd.Happened() {
		_ = statusCommand(ctx)
	} else if initCmd.Happened() {
		initCommand()
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

func exitWithError(message string) {
	fmt.Println(message)
	os.Exit(1)
}
