package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CredentialStore struct {
	key     []byte
	filePath string
}

func NewCredentialStore(dataDir string) *CredentialStore {
	key := deriveKey(dataDir)
	return &CredentialStore{
		key:     key,
		filePath: filepath.Join(dataDir, "auth.json.enc"),
	}
}

func deriveKey(seed string) []byte {
	h := sha256.Sum256([]byte(seed + "-codeagent-credentials"))
	return h[:]
}

func (s *CredentialStore) SaveCredentials(creds map[string]string) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	encrypted, err := s.encrypt(data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(s.filePath, []byte(encrypted), 0600)
}

func (s *CredentialStore) LoadCredentials() (map[string]string, error) {
	encrypted, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	decrypted, err := s.decrypt(string(encrypted))
	if err != nil {
		// Fallback to unencrypted file
		return s.loadLegacyCredentials()
	}

	var creds map[string]string
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		return nil, err
	}

	return creds, nil
}

func (s *CredentialStore) loadLegacyCredentials() (map[string]string, error) {
	legacyPath := filepath.Join(filepath.Dir(s.filePath), "auth.json")
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	// Migrate to encrypted storage
	s.SaveCredentials(creds)
	os.Remove(legacyPath)

	return creds, nil
}

func (s *CredentialStore) encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *CredentialStore) decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
