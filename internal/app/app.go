package app

import (
	"context"
	"errors"
	"fmt"
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
	timeout       time.Duration
	identifier    string
	sessionMode   SessionPersistenceMode
	sessionTokens *api.AuthenticationTokensResponse
}

func New(token, env, identifier string, opts ...Option) (*App, error) {

	a := &App{
		timeout:     30 * time.Second,
		identifier:  identifier,
		sessionMode: SessionPersistAuto,
	}

	for _, opt := range opts {
		opt(a)
	}

	httpClient := &http.Client{
		Timeout: a.timeout,
	}

	err := a.Env.UnmarshalText([]byte(env))
	if err != nil {
		logger.Errorf("error parsing environment %q: %v", env, err)
		return nil, fmt.Errorf("parse environment %q: %w", env, err)
	}

	a.AuthFacade, err = ksef.NewAuthFacade(a.Env, httpClient)

	if err != nil {
		logger.Errorf("error creating auth facade: %v", err)
		return nil, fmt.Errorf("error creating auth facade %w", err)
	}

	a.Encryptor, err = ksef.NewEncryptionService(a.Env, httpClient)
	if err != nil {
		logger.Errorf("error creating Encryption Service: %v", err)
		return nil, fmt.Errorf("error creating Encryption Service %w", err)
	}

	a.TokenProvider = ksef.NewTokenProvider(a.AuthFacade, func(ctx context.Context) (*api.AuthenticationTokensResponse, error) {
		authToken, err := ResolveAuthToken(token, env, a.identifier)
		if err != nil {
			return nil, fmt.Errorf("full authentication requires KSeF authorisation token: %w", err)
		}
		return ksef.WithKsefToken(ctx, a.AuthFacade, a.Encryptor, authToken)
	})

	if a.sessionMode == SessionPersistAuto {
		a.TokenProvider.SetTokenUpdateCallback(func(ctx context.Context, update ksef.TokenUpdate) error {
			t := &api.AuthenticationTokensResponse{
				AccessToken:  update.AccessToken,
				RefreshToken: update.RefreshToken,
			}

			if err := StoreSessionTokenWithError(t, env, update.NIP); err != nil {
				return fmt.Errorf("store updated session token: %w", err)
			}
			return nil
		})
		logger.Info("session persistence enabled")
	}

	if a.sessionTokens == nil && a.identifier != "" {
		storedTokens, err := TryLoadSessionToken(env, a.identifier)
		if err != nil {
			if errors.Is(err, ErrEncryptionKeyNotInitialized) {
				return nil, fmt.Errorf("cannot read stored session token: encryption key is not initialized; run: ksef-cli init")
			}
			if errors.Is(err, ErrInvalidEncryptionKey) {
				return nil, fmt.Errorf("cannot read stored session token: encryption key in keystore is invalid; run: ksef-cli init --force-init and login again")
			}
			logger.Errorf("error loading stored session token: %v", err)
			return nil, fmt.Errorf("load stored session token: %w", err)
		}
		a.sessionTokens = storedTokens
		logger.Info("session tokens loaded")
	}

	if a.sessionTokens != nil {
		seedCtx := ksef.ContextWithEnv(context.Background(), a.identifier, a.Env)
		err := a.TokenProvider.SeedTokens(seedCtx, a.sessionTokens.AccessToken, a.sessionTokens.RefreshToken)
		if err != nil {
			logger.Errorf("error Seed Tokens in TokenProvider: %v", err)
			return nil, fmt.Errorf("error seed tokens in TokenProvider %w", err)
		}
		logger.Info("session tokens seeded")
	}

	a.Client, err = ksef.NewClient(a.Env, httpClient, a.TokenProvider)
	if err != nil {
		logger.Errorf("error creating KSeF client: %v", err)
		return nil, fmt.Errorf("create KSeF client: %w", err)
	}

	return a, nil
}

type Option func(*App)

type SessionPersistenceMode uint8

const (
	SessionPersistAuto SessionPersistenceMode = iota
	SessionPersistDisabled
)

func WithTimeout(timeout time.Duration) Option {
	return func(c *App) {
		c.timeout = timeout
	}
}

func WithSessionPersistence(mode SessionPersistenceMode) Option {
	return func(c *App) {
		c.sessionMode = mode
	}
}
