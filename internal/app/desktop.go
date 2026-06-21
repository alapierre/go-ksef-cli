package app

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const service = "ksef-cli"
const user = "encryption-key"

var (
	ErrEncryptionKeyNotInitialized = errors.New("encryption key is not initialized")
	ErrInvalidEncryptionKey        = errors.New("invalid encryption key in keystore")
)

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
		return fmt.Errorf("save encryption key in keystore: %w", err)
	}

	return nil
}

func loadEncryptionKey() ([]byte, error) {
	secret, err := keyring.Get(service, user)
	if err != nil {
		if isKeyringNotFound(err) {
			return nil, ErrEncryptionKeyNotInitialized
		}
		return nil, fmt.Errorf("read encryption key from keystore: %w", err)
	}

	key, err := base64.RawStdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEncryptionKey, err)
	}

	return key, nil
}

func isKeyringNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}
