package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	authTokenFilePrefix    = ".authorisation_token_"
	sessionTokenFilePrefix = ".session_token_"
)

type TokenStorageStatus struct {
	Env        string
	Identifier string

	HasAuthToken bool
	AuthTokenErr error

	HasSessionTokens bool
	SessionTokenErr  error

	AccessValidUntil  time.Time
	RefreshValidUntil time.Time
	AccessValid       bool
	RefreshValid      bool
}

func (s TokenStorageStatus) IsLoggedIn() bool {
	return s.HasSessionTokens && s.SessionTokenErr == nil && s.RefreshValid
}

func ListTokenStatuses(env string, all bool) ([]TokenStorageStatus, error) {
	envs, err := listEnvs(env, all)
	if err != nil {
		return nil, err
	}

	var out []TokenStorageStatus
	for _, e := range envs {
		records, err := listTokenStatusForEnv(e)
		if err != nil {
			return nil, err
		}
		out = append(out, records...)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Env == out[j].Env {
			return out[i].Identifier < out[j].Identifier
		}
		return out[i].Env < out[j].Env
	})
	return out, nil
}

func listTokenStatusForEnv(env string) ([]TokenStorageStatus, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".go-ksef-cli", env)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read token directory %s: %w", dir, err)
	}

	byIdentifier := make(map[string]*TokenStorageStatus)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		switch {
		case strings.HasPrefix(name, authTokenFilePrefix):
			identifier := strings.TrimPrefix(name, authTokenFilePrefix)
			s := ensureStatus(byIdentifier, env, identifier)
			s.HasAuthToken = true
		case strings.HasPrefix(name, sessionTokenFilePrefix):
			identifier := strings.TrimPrefix(name, sessionTokenFilePrefix)
			s := ensureStatus(byIdentifier, env, identifier)
			s.HasSessionTokens = true
		}
	}

	now := time.Now().UTC()
	out := make([]TokenStorageStatus, 0, len(byIdentifier))
	for _, s := range byIdentifier {
		if s.HasAuthToken {
			if _, err := LoadAuthToken(s.Env, s.Identifier); err != nil {
				s.AuthTokenErr = err
			}
		}

		if s.HasSessionTokens {
			tokens, err := TryLoadSessionToken(s.Env, s.Identifier)
			if err != nil {
				s.SessionTokenErr = err
			} else if tokens != nil {
				s.AccessValidUntil = tokens.AccessToken.GetValidUntil().UTC()
				s.RefreshValidUntil = tokens.RefreshToken.GetValidUntil().UTC()
				s.AccessValid = s.AccessValidUntil.After(now)
				s.RefreshValid = s.RefreshValidUntil.After(now)
			}
		}

		out = append(out, *s)
	}
	return out, nil
}

func ensureStatus(byIdentifier map[string]*TokenStorageStatus, env, identifier string) *TokenStorageStatus {
	s, ok := byIdentifier[identifier]
	if ok {
		return s
	}

	s = &TokenStorageStatus{
		Env:        env,
		Identifier: identifier,
	}
	byIdentifier[identifier] = s
	return s
}

func listEnvs(env string, all bool) ([]string, error) {
	if !all {
		return []string{strings.ToUpper(strings.TrimSpace(env))}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	base := filepath.Join(home, ".go-ksef-cli")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read token base directory %s: %w", base, err)
	}

	envs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, strings.ToUpper(strings.TrimSpace(entry.Name())))
		}
	}

	sort.Strings(envs)
	return envs, nil
}
