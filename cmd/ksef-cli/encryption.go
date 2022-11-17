package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/alapierre/go-ksef-client/ksef/cipher"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/zalando/go-keyring"
	"os"
	"path/filepath"
	"strings"
)

const service = "ksef-cli"
const user = "encryption-key"

func initEncryption() error {

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("AES random key creation error, %v", err)
	}

	err := keyring.Set(service, user, string(key))
	if err != nil {
		logrus.Fatal(err)
	}

	return nil
}

func loadEncryptionKey() []byte {
	secret, err := keyring.Get(service, user)
	if err != nil {
		logrus.Fatal(err)
	}
	return []byte(secret)
}

func checkIsEncKeyInitialized() bool {
	_, err := keyring.Get(service, user)
	return err == nil
}

func crypt(message string) (string, error) {

	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("AES random iv creation error, %v", err)
	}

	aes, err := cipher.NewAes(loadEncryptionKey(), iv)
	if err != nil {
		return "", err
	}

	encrypted, err := aes.Encrypt([]byte(message))
	if err != nil {
		return "", err
	}

	concat := append(iv[:], encrypted[:]...)

	return base64.StdEncoding.EncodeToString(concat), nil
}

func decrypt(message string) (string, error) {

	decoded, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return "", err
	}

	iv := decoded[:16]
	encrypted := decoded[16:]

	aes, err := cipher.NewAes(loadEncryptionKey(), iv)
	if err != nil {
		return "", err
	}

	plainText, err := aes.Decrypt(encrypted)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

func storeAuthToken(token, nip, env string) {
	file := fmt.Sprintf(".authorisation_token_%s", nip)
	storeToken(token, file, env)
}

func loadAuthToken(nip, env string) string {
	file := fmt.Sprintf(".authorisation_token_%s", nip)
	return loadToken(file, env)
}

func storeSessionToken(token, env string) {
	storeToken(token, ".session_token", env)
}

func loadSessionToken(env string) string {
	return loadToken(".session_token", env)
}

func storeToken(token, fileName, env string) {

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	encrypted, err := crypt(token)
	if err != nil {
		log.Fatal(err)
	}

	file := filepath.Join(path, fileName)
	err = os.WriteFile(file, []byte(encrypted), 0600)
	if err != nil {
		log.Fatal(err)
	}
}

func loadToken(fileName, env string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	path := filepath.Join(home, ".go-ksef-cli", strings.ToUpper(env))
	file := filepath.Join(path, fileName)

	if _, err := os.Stat(file); err != nil {
		fmt.Printf("Stored token not exist for env %s", strings.ToUpper(env))
		os.Exit(1)
	}

	token, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	result, err := decrypt(string(token))
	if err != nil {
		log.Fatal(err)
	}
	return result
}
