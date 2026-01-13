package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alapierre/go-ksef-client/ksef/api"
	"github.com/zalando/go-keyring"
)

const service = "ksef-cli"
const user = "encryption-key"

func initEncryption() error {

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("AES random key creation error, %v", err)
	}

	keyB64 := base64.RawStdEncoding.EncodeToString(key)
	err := keyring.Set(service, user, keyB64)
	if err != nil {
		logger.Fatal(err)
	}

	return nil
}

func loadEncryptionKey() []byte {
	secret, err := keyring.Get(service, user)
	if err != nil {
		logger.Fatal(err)
	}

	key, err := base64.RawStdEncoding.DecodeString(secret)
	if err != nil {
		logger.Fatalf("invalid key encoding in keyring: %+v", err)
	}

	return key
}

func checkIsEncKeyInitialized() bool {
	_, err := keyring.Get(service, user)
	return err == nil
}

func storeAuthToken(token, nip, env string) {
	file := fmt.Sprintf(".authorisation_token_%s", nip)
	storeToken([]byte(token), file, env, nip)
}

func loadAuthToken(nip, env string) string {
	file := fmt.Sprintf(".authorisation_token_%s", nip)
	return string(loadToken(file, env, nip))
}

func storeSessionToken(token *api.AuthenticationTokensResponse, env, nip string) {
	b, err := encodeGob(token)
	if err != nil {
		return
	}
	storeToken(b, ".session_token", env, nip)
}

func loadSessionToken(env, nip string) *api.AuthenticationTokensResponse {

	b := loadToken(".session_token", env, nip)
	var token api.AuthenticationTokensResponse

	if err := decodeGob(b, &token); err != nil {
		logger.Fatal(err)
	}
	return &token
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

	encrypted, err := EncryptAESGCM(loadEncryptionKey(), token, buildAAD(env, nip))
	if err != nil {
		logger.Fatal(err)
	}

	file := filepath.Join(path, fileName)
	err = os.WriteFile(file, encrypted, 0600)
	if err != nil {
		logger.Fatal(err)
	}
}

func buildAAD(env, nip string) []byte {
	env = strings.ToUpper(strings.TrimSpace(env))
	logger.Debugf("Building AAD for env: %s, nip: %s", env, nip)
	return []byte("go-ksef-cli" + "\x00" + env + "\x00" + nip)
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

	result, err := DecryptAESGCM(loadEncryptionKey(), token, buildAAD(env, nip))
	if err != nil {
		logger.Fatal(err)
	}
	return result
}

func EncryptAESGCM(key []byte, plaintext []byte, aad []byte) ([]byte, error) {
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

func DecryptAESGCM(key []byte, data []byte, aad []byte) ([]byte, error) {
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
