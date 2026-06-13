package app

import (
	"context"
	"net/http"
	"time"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "app")

type App struct {
	PathToKey string

	Client        *ksef.Client
	TokenProvider *ksef.TokenProvider
	Encryptor     *ksef.EncryptionService
	AuthFacade    *ksef.AuthFacade
	Env           ksef.Environment
}

func New(token, env string) *App {

	// jeśli nie ma tokena, to załadować ze store
	// to będzie wywoływane w dwóch kontekstach
	// — dla przeprowadzenia pełnego uwierzytelnienia tokenem w ksef w przypadku polecenia login
	// — pozostałych przypadkach, gdy para tokenów jest zapisana. Wtedy funkcja authenticator będzie inna — otrzyma gotową parę, jeśli refresh jest ważny
	// — alternatywnie, jeśli refresh jest nie ważny, to może przeprowadzić pełne uwierzytelnienie

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	var c App

	err := c.Env.UnmarshalText([]byte(env))
	c.AuthFacade, err = ksef.NewAuthFacade(c.Env, httpClient)

	if err != nil {
		logger.Fatal(err)
	}

	c.Encryptor, err = ksef.NewEncryptionService(c.Env, httpClient)
	if err != nil {
		logger.Fatal(err)
	}

	c.TokenProvider = ksef.NewTokenProvider(c.AuthFacade, func(ctx context.Context) (*api.AuthenticationTokensResponse, error) {
		return ksef.WithKsefToken(ctx, c.AuthFacade, c.Encryptor, token)
	})

	c.Client, err = ksef.NewClient(c.Env, httpClient, c.TokenProvider)
	return &c
}
