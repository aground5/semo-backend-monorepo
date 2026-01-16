package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

type EncryptionService interface {
	Encrypt(plaintext string) (ciphertext, iv string, err error)
	Decrypt(ciphertext, iv string) (plaintext string, err error)
}

type AESEncryptionService struct {
	key []byte
}

func NewAESEncryptionService(hexKey string) (*AESEncryptionService, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, errors.New("invalid encryption key format")
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes (64 hex chars)")
	}
	return &AESEncryptionService{key: key}, nil
}

func (s *AESEncryptionService) Encrypt(plaintext string) (string, string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", "", err
	}

	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(iv),
		nil
}

func (s *AESEncryptionService) Decrypt(ciphertextB64, ivB64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	iv, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
