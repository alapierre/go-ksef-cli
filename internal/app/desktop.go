package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "ksef-cli"
const user = "encryption-key"

func CheckIsEncKeyInitialized() bool {
	_, err := keyring.Get(service, user)
	return err == nil
}

func InitEncryption() error {

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
