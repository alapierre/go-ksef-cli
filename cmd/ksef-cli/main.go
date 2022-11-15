package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/alapierre/go-ksef-client/ksef/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
	"os"
)

type context struct {
	pathToKey string
	env       string
}

var c context

func main() {

	parser := argparse.NewParser("ksef-cli", "KSeF Command line interface")

	loginCmd := parser.NewCommand("login", "login into KSeF using provided authorisation token")
	token := loginCmd.String("t", "token", &argparse.Options{Required: true, Help: "KSeF authorisation token"})
	identifier := loginCmd.String("i", "identifier", &argparse.Options{Required: true, Help: "Organization identifier (NIP)"})

	config()

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if loginCmd.Happened() {
		loginCommand(token, identifier, &c.pathToKey)
	}

}

func loginCommand(token *string, identifier, pathToKey *string) {

	fmt.Println("Trying to login into KSeF")

	client := api.New(stringToEnvName(c.env))
	session := api.NewSessionService(client)

	sessionToken, err := session.LoginByToken(*identifier, model.ONIP, *token, *pathToKey)

	if err != nil {
		re, ok := err.(*api.RequestError)
		if ok {
			log.Errorf("request error %d responce body %s", re.StatusCode, re.Body)
		}
		panic(err)
	}

	fmt.Printf("session token: %s\n", sessionToken.SessionToken.Token)
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
		panic(fmt.Errorf("fatal error config file: %w", err))
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

func storeKeyInKeyring() {
	service := "ksef-cli"
	user := "encryption-key"
	password := "encryption-key"

	// set password
	err := keyring.Set(service, user, password)
	if err != nil {
		log.Fatal(err)
	}

	// get password
	secret, err := keyring.Get(service, user)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(secret)
}
