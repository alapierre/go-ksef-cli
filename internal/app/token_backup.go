package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alapierre/go-ksef-client/ksef/api"
	"golang.org/x/crypto/pbkdf2"
)

const (
	tokenBackupVersion      = 1
	tokenBackupSaltSize     = 16
	tokenBackupPBKDF2Rounds = 200_000
	tokenBackupAAD          = "go-ksef-cli-token-backup-v1"
)

var (
	ErrInvalidTokenBackupPassword = errors.New("invalid token backup password")
)

type TokenBackup struct {
	Version   int                `json:"version"`
	CreatedAt time.Time          `json:"created_at"`
	Entries   []TokenBackupEntry `json:"entries"`
}

type TokenBackupEntry struct {
	Env           string                            `json:"env"`
	Identifier    string                            `json:"identifier"`
	AuthToken     string                            `json:"auth_token,omitempty"`
	SessionTokens *api.AuthenticationTokensResponse `json:"session_tokens,omitempty"`
}

type EncryptedTokenBackup struct {
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	SaltB64    string `json:"salt_b64"`
	NonceB64   string `json:"nonce_b64"`
	DataB64    string `json:"data_b64"`
}

func BuildTokenBackup(env string, all bool, identifier string) (*TokenBackup, error) {
	backup, _, err := BuildTokenBackupBestEffort(env, all, identifier)
	return backup, err
}

func BuildTokenBackupBestEffort(env string, all bool, identifier string) (*TokenBackup, []string, error) {
	statuses, err := ListTokenStatuses(env, all)
	if err != nil {
		return nil, nil, err
	}

	filterIdentifier := strings.TrimSpace(identifier)
	entries := make([]TokenBackupEntry, 0, len(statuses))
	warnings := make([]string, 0)
	for _, s := range statuses {
		if filterIdentifier != "" && s.Identifier != filterIdentifier {
			continue
		}

		entry := TokenBackupEntry{
			Env:        s.Env,
			Identifier: s.Identifier,
		}

		if s.HasAuthToken {
			authToken, err := LoadAuthToken(s.Env, s.Identifier)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("skipping auth token for env %s identifier %s: %v", s.Env, s.Identifier, err))
			} else {
				entry.AuthToken = authToken
			}
		}

		if s.HasSessionTokens {
			sessionTokens, err := TryLoadSessionToken(s.Env, s.Identifier)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("skipping session tokens for env %s identifier %s: %v", s.Env, s.Identifier, err))
			} else {
				entry.SessionTokens = sessionTokens
			}
		}

		if entry.AuthToken == "" && entry.SessionTokens == nil {
			continue
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, warnings, fmt.Errorf("no exportable tokens found")
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Env == entries[j].Env {
			return entries[i].Identifier < entries[j].Identifier
		}
		return entries[i].Env < entries[j].Env
	})

	return &TokenBackup{
		Version:   tokenBackupVersion,
		CreatedAt: time.Now().UTC(),
		Entries:   entries,
	}, warnings, nil
}

func ImportTokenBackup(backup *TokenBackup) error {
	if backup == nil {
		return fmt.Errorf("token backup is nil")
	}
	if backup.Version != tokenBackupVersion {
		return fmt.Errorf("unsupported token backup version: %d", backup.Version)
	}
	if len(backup.Entries) == 0 {
		return fmt.Errorf("token backup has no entries")
	}

	for _, entry := range backup.Entries {
		if strings.TrimSpace(entry.Env) == "" || strings.TrimSpace(entry.Identifier) == "" {
			return fmt.Errorf("invalid token backup entry: missing env or identifier")
		}

		if entry.AuthToken != "" {
			if err := StoreAuthToken([]byte(entry.AuthToken), entry.Env, entry.Identifier); err != nil {
				return fmt.Errorf("store auth token for env %s identifier %s: %w", entry.Env, entry.Identifier, err)
			}
		}

		if entry.SessionTokens != nil {
			if err := StoreSessionTokenWithError(entry.SessionTokens, entry.Env, entry.Identifier); err != nil {
				return fmt.Errorf("store session tokens for env %s identifier %s: %w", entry.Env, entry.Identifier, err)
			}
		}
	}

	return nil
}

func EncryptTokenBackup(backup *TokenBackup, password string) ([]byte, error) {
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("backup password cannot be empty")
	}
	if backup == nil {
		return nil, fmt.Errorf("token backup is nil")
	}

	plain, err := json.Marshal(backup)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, tokenBackupSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key := deriveBackupKey(password, salt, tokenBackupPBKDF2Rounds)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	encrypted := gcm.Seal(nil, nonce, plain, []byte(tokenBackupAAD))

	envelope := EncryptedTokenBackup{
		Version:    tokenBackupVersion,
		KDF:        "pbkdf2-sha256",
		Iterations: tokenBackupPBKDF2Rounds,
		SaltB64:    base64.RawStdEncoding.EncodeToString(salt),
		NonceB64:   base64.RawStdEncoding.EncodeToString(nonce),
		DataB64:    base64.RawStdEncoding.EncodeToString(encrypted),
	}

	return json.MarshalIndent(envelope, "", "  ")
}

func DecryptTokenBackup(data []byte, password string) (*TokenBackup, error) {
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("backup password cannot be empty")
	}

	var envelope EncryptedTokenBackup
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode encrypted token backup: %w", err)
	}

	if envelope.Version != tokenBackupVersion {
		return nil, fmt.Errorf("unsupported encrypted token backup version: %d", envelope.Version)
	}
	if envelope.KDF != "pbkdf2-sha256" {
		return nil, fmt.Errorf("unsupported encrypted token backup KDF: %s", envelope.KDF)
	}
	if envelope.Iterations <= 0 {
		return nil, fmt.Errorf("invalid encrypted token backup iteration count: %d", envelope.Iterations)
	}

	salt, err := base64.RawStdEncoding.DecodeString(envelope.SaltB64)
	if err != nil {
		return nil, fmt.Errorf("decode backup salt: %w", err)
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.NonceB64)
	if err != nil {
		return nil, fmt.Errorf("decode backup nonce: %w", err)
	}
	encrypted, err := base64.RawStdEncoding.DecodeString(envelope.DataB64)
	if err != nil {
		return nil, fmt.Errorf("decode backup payload: %w", err)
	}

	key := deriveBackupKey(password, salt, envelope.Iterations)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plain, err := gcm.Open(nil, nonce, encrypted, []byte(tokenBackupAAD))
	if err != nil {
		return nil, ErrInvalidTokenBackupPassword
	}

	var backup TokenBackup
	if err := json.Unmarshal(plain, &backup); err != nil {
		return nil, fmt.Errorf("decode token backup: %w", err)
	}

	return &backup, nil
}

func deriveBackupKey(password string, salt []byte, rounds int) []byte {
	return pbkdf2.Key([]byte(password), salt, rounds, 32, sha256.New)
}
