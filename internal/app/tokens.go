package app

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alapierre/go-ksef-client/ksef/api"
)

var (
	ErrSessionTokenNotFound = errors.New("stored session token not found")
	ErrAuthTokenNotFound    = errors.New("stored authorisation token not found")
	ErrAuthTokenDecrypt     = errors.New("stored authorisation token cannot be decrypted")
)

func StoreSessionToken(token *api.AuthenticationTokensResponse, env, nip string) {
	if err := StoreSessionTokenWithError(token, env, nip); err != nil {
		logger.Errorf("cannot store session token: %v", err)
	}
}

func LoadSessionToken(env, nip string) (*api.AuthenticationTokensResponse, error) {
	token, err := TryLoadSessionToken(env, nip)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("%w for env %s and identifier %s", ErrSessionTokenNotFound, strings.ToUpper(strings.TrimSpace(env)), nip)
	}

	return token, nil
}

func TryLoadSessionToken(env, nip string) (*api.AuthenticationTokensResponse, error) {
	env = strings.ToUpper(strings.TrimSpace(env))
	file := fmt.Sprintf(".session_token_%s", nip)
	b, err := loadToken(file, env, nip)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var token api.AuthenticationTokensResponse

	if err := decodeGob(b, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func StoreSessionTokenWithError(token *api.AuthenticationTokensResponse, env, nip string) error {
	env = strings.ToUpper(strings.TrimSpace(env))
	b, err := encodeGob(token)
	if err != nil {
		return err
	}
	file := fmt.Sprintf(".session_token_%s", nip)
	return storeToken(b, file, env, nip)
}

func DeleteSessionToken(env, nip string) error {
	env = strings.ToUpper(strings.TrimSpace(env))
	fileName := fmt.Sprintf(".session_token_%s", nip)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(home, ".go-ksef-cli", env, fileName)
	err = os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ResolveAuthToken(token, env, nip string) (string, error) {
	token = strings.TrimSpace(token)
	if token != "" {
		return token, nil
	}

	logger.Infof("Token not provided, trying to read from file")

	storedToken, err := LoadAuthToken(env, nip)
	if err != nil {
		if errors.Is(err, ErrEncryptionKeyNotInitialized) {
			return "", fmt.Errorf("encryption key is not initialized; run: ksef-cli init: %w", err)
		}
		if errors.Is(err, ErrInvalidEncryptionKey) {
			return "", fmt.Errorf("encryption key in keystore is invalid; run: ksef-cli init --force-init and store token again: %w", err)
		}
		if errors.Is(err, ErrAuthTokenDecrypt) {
			return "", fmt.Errorf("stored auth token cannot be decrypted; run: ksef-cli --env %s store --identifier %s --token \"...\": %w", strings.ToUpper(strings.TrimSpace(env)), nip, err)
		}
		if errors.Is(err, ErrAuthTokenNotFound) {
			return "", fmt.Errorf("token not provided; pass --token (or KSEF_TOKEN) or store it first using: ksef-cli --env %s store --identifier %s --token \"...\": %w", strings.ToUpper(strings.TrimSpace(env)), nip, err)
		}
		return "", fmt.Errorf("token not provided and cannot read stored token: %w", err)
	}

	return storedToken, nil
}

func loadToken(fileName, env, nip string) ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	file := filepath.Join(path, fileName)

	token, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return nil, err
	}

	result, err := decryptAESGCM(key, token, buildAAD(env, nip))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func decryptAESGCM(key []byte, data []byte, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ns := gcm.NonceSize()
	if len(data) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce := data[:ns]
	ciphertext := data[ns:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func encodeGob(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeGob(data []byte, v any) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(v)
}

func storeToken(token []byte, fileName, env, nip string) error {

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return err
	}

	encrypted, err := encryptAESGCM(key, token, buildAAD(env, nip))
	if err != nil {
		return err
	}

	file := filepath.Join(path, fileName)
	err = os.WriteFile(file, encrypted, 0600)
	if err != nil {
		return err
	}
	return nil
}

func StoreAuthToken(token []byte, env, nip string) error {
	fileName := fmt.Sprintf(".authorisation_token_%s", nip)
	return storeToken(token, fileName, env, nip)
}

func LoadAuthToken(env, nip string) (string, error) {

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Error("cannot load home directory", err)
		return "", err
	}

	fileName := fmt.Sprintf(".authorisation_token_%s", nip)
	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	file := filepath.Join(path, fileName)
	data, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w for env %s and identifier %s", ErrAuthTokenNotFound, strings.ToUpper(strings.TrimSpace(env)), nip)
		}
		return "", fmt.Errorf("cannot read stored auth token file %s: %w", file, err)
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	decrypted, err := decryptAESGCM(key, data, buildAAD(env, nip))

	if err != nil {
		return "", fmt.Errorf("%w (%s): %w", ErrAuthTokenDecrypt, file, err)
	}

	return string(decrypted), nil
}

func encryptAESGCM(key []byte, plaintext []byte, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
	out := append(nonce, ciphertext...)
	return out, nil
}

func buildAAD(env, nip string) []byte {
	env = strings.ToUpper(strings.TrimSpace(env))
	logger.Debugf("Building AAD for env: %s, nip: %s", env, nip)
	return []byte("go-ksef-cli" + "\x00" + env + "\x00" + nip)
}
