package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/alapierre/go-ksef-client/ksef/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

type context struct {
	pathToKey      string
	env            string
	sessionService api.SessionService
}

var c context

func main() {

	parser := argparse.NewParser("ksef-cli", "KSeF Command line interface")

	loginCmd := parser.NewCommand("login", "login into KSeF using provided authorisation token")
	token := loginCmd.String("t", "token", &argparse.Options{Required: true, Help: "KSeF authorisation token"})
	identifier := loginCmd.String("i", "identifier", &argparse.Options{Required: true, Help: "Organization identifier (NIP)"})

	logoutCmd := parser.NewCommand("logout", "logout from KSeF by close interactive session")
	sessionToken := logoutCmd.String("t", "token", &argparse.Options{Required: true, Help: "KSeF session token"})

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
	}

}

func loginCommand(token, identifier, pathToKey *string) {

	fmt.Println("Trying to login into KSeF")

	sessionToken, err := c.sessionService.LoginByToken(*identifier, model.ONIP, *token, *pathToKey)
	handleError(err)
	fmt.Printf("session token: %s\n", sessionToken.SessionToken.Token)
}

func logoutCommand(sessionToken *string) {
	terminate, err := c.sessionService.Terminate(*sessionToken)
	handleError(err)
	fmt.Printf("responce: %#v\n", *terminate)
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

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Can't load %s\n", err)
	}

	keysPath := viper.GetString("mfKeys")
	ksefEnv := viper.GetString("env")
	c.pathToKey = fmt.Sprintf("%s/%s/publicKey.pem", keysPath, ksefEnv)

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
