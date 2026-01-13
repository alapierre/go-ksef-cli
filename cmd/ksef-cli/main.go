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
	pathToKey string

	client        *ksef.Client
	tokenProvider *ksef.TokenProvider
	encryptor     *ksef.EncryptionService
	authFacade    *ksef.AuthFacade
	env           ksef.Environment
}

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
	forceInit := initCmd.Flag("f", "force", &argparse.Options{Help: "force initialization even if key is already initialized", Default: false, Required: false})

	storeAuthTokenCmd := parser.NewCommand("store", "encrypt and store authorisation token in keystore selected in configuration")
	tokenToStore := storeAuthTokenCmd.String("t", "token", &argparse.Options{Required: true, Help: "KSeF authorisation token"})

	statusCmd := parser.NewCommand("status", "print KSeF session status")
	//statusToken := statusCmd.String("t", "token", &argparse.Options{Required: false, Help: "KSeF session token, if not provided it will be loaded (you should login first"})

	config()

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	appContext := prepareAppContext(*token, viper.GetString("env"))
	ctx := ksef.ContextWithEnv(context.Background(), *identifier, appContext.env)

	if loginCmd.Happened() {
		loginCommand(ctx, *token, appContext)
		//} else if logoutCmd.Happened() {
		//	logoutCommand(sessionToken)
		//} else if sendInvoiceCmd.Happened() {
		//	sendCommand(*sendToken, fileToSend)
	} else if storeAuthTokenCmd.Happened() {
		storeAuthToken(*tokenToStore, *identifier, appContext.env.Name())
		fmt.Printf("Token for identifier %s for environment: %s stored successfully\n", *identifier, appContext.env.Name())
	} else if statusCmd.Happened() {
		_ = statusCommand(ctx)
	} else if initCmd.Happened() {
		initCommand(*forceInit)
	}
}

func prepareAppContext(token string, envStr string) *app {

	// jeśli nie ma tokena, to załadować ze store
	// to będzie wywoływane w dwóch kontekstach
	// — dla przeprowadzenia pełnego uwierzytelnienia tokenem w ksef w przypadku polecenia login
	// — pozostałych przypadkach, gdy para tokenów jest zapisana. Wtedy funkcja authenticator będzie inna — otrzyma gotową parę, jeśli refresh jest ważny
	// — alternatywnie, jeśli refresh jest nie ważny, to może przeprowadzić pełne uwierzytelnienie

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	var c app

	err := c.env.UnmarshalText([]byte(envStr))
	c.authFacade, err = ksef.NewAuthFacade(c.env, httpClient)

	if err != nil {
		logger.Fatal(err)
	}

	c.encryptor, err = ksef.NewEncryptionService(c.env, httpClient)
	if err != nil {
		logger.Fatal(err)
	}

	c.tokenProvider = ksef.NewTokenProvider(c.authFacade, func(ctx context.Context) (*api.AuthenticationTokensResponse, error) {
		return ksef.WithKsefToken(ctx, c.authFacade, c.encryptor, token)
	})

	c.client, err = ksef.NewClient(c.env, httpClient, c.tokenProvider)
	return &c
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

}

func exitWithError(message string) {
	fmt.Println(message)
	os.Exit(1)
}
