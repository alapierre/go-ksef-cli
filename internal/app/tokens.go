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

func StoreSessionToken(token *api.AuthenticationTokensResponse, env, nip string) {
	env = strings.ToUpper(strings.TrimSpace(env))
	b, err := encodeGob(token)
	if err != nil {
		return
	}
	file := fmt.Sprintf(".session_token_%s", nip)
	storeToken(b, file, env, nip)
}

func LoadSessionToken(env, nip string) *api.AuthenticationTokensResponse {

	env = strings.ToUpper(strings.TrimSpace(env))
	file := fmt.Sprintf(".session_token_%s", nip)
	b := loadToken(file, env, nip)
	var token api.AuthenticationTokensResponse

	if err := decodeGob(b, &token); err != nil {
		logger.Fatal(err)
	}
	return &token
}

func ResolveAuthToken(token, env, nip string) (string, error) {
	token = strings.TrimSpace(token)
	if token != "" {
		return token, nil
	}

	logger.Infof("Token not provided, trying to read from file")

	storedToken, err := LoadAuthToken(env, nip)
	if err != nil {
		return "", fmt.Errorf("token not provided and cannot read stored token: %w", err)
	}

	return storedToken, nil
}

func loadToken(fileName, env, nip string) []byte {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal(err)
	}

	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	file := filepath.Join(path, fileName)

	if _, err := os.Stat(file); err != nil {
		fmt.Printf("Stored token not exist for env %s", strings.ToUpper(env))
		os.Exit(1)
	}

	token, err := os.ReadFile(file)
	if err != nil {
		logger.Fatal(err)
	}

	result, err := decryptAESGCM(loadEncryptionKey(), token, buildAAD(env, nip))
	if err != nil {
		logger.Fatal(err)
	}
	return result
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

func storeToken(token []byte, fileName, env, nip string) {

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal(err)
	}

	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		logger.Fatal(err)
	}

	encrypted, err := encryptAESGCM(loadEncryptionKey(), token, buildAAD(env, nip))
	if err != nil {
		logger.Fatal(err)
	}

	file := filepath.Join(path, fileName)
	err = os.WriteFile(file, encrypted, 0600)
	if err != nil {
		logger.Fatal(err)
	}
}

func StoreAuthToken(token []byte, env, nip string) {
	fileName := fmt.Sprintf(".authorisation_token_%s", nip)
	storeToken(token, fileName, env, nip)
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
		logger.Error("cannot read file", err)
		return "", err
	}

	decrypted, err := decryptAESGCM(loadEncryptionKey(), data, buildAAD(env, nip))

	if err != nil {
		logger.Errorf("cannot decrypt stored auth token (%s): %v", path, err)
		return "", err
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
