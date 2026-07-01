package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// sessionData is the content stored inside the encrypted httpOnly session cookie.
// It carries the Identity Platform refresh token alongside minimal user info and
// the current ID token so the middleware can verify and transparently refresh it.
//
// The session is encrypted with AES-256-GCM; the cookie value is the base64url
// encoding of nonce‖ciphertext. No server-side store is used — the cookie is
// the session (single-user; nothing to persist on the server).
type sessionData struct {
	UID          string `json:"uid"`
	Email        string `json:"email"`
	RefreshToken string `json:"rt"`
	IDToken      string `json:"idt"`
	ExpiresAt    int64  `json:"exp"` // Unix timestamp of ID token expiry
}

// encodeSession serialises and AES-256-GCM-encrypts a sessionData value into a
// base64url-encoded string suitable for use as an httpOnly cookie value.
func encodeSession(key []byte, s *sessionData) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshalling session: %w", err)
	}
	ct, err := aesgcmEncrypt(key, payload)
	if err != nil {
		return "", fmt.Errorf("encrypting session: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(ct), nil
}

// decodeSession base64url-decodes and AES-256-GCM-decrypts a cookie value produced
// by [encodeSession], returning the original sessionData.
func decodeSession(key []byte, value string) (*sessionData, error) {
	ct, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("base64-decoding session cookie: %w", err)
	}
	payload, err := aesgcmDecrypt(key, ct)
	if err != nil {
		return nil, fmt.Errorf("decrypting session cookie: %w", err)
	}
	var s sessionData
	if err := json.Unmarshal(payload, &s); err != nil {
		return nil, fmt.Errorf("unmarshalling session: %w", err)
	}
	return &s, nil
}

// sessionExpiresAt returns the time.Time at which the ID token stored in the session
// is due to expire.
func sessionExpiresAt(s *sessionData) time.Time {
	return time.Unix(s.ExpiresAt, 0)
}

// aesgcmEncrypt encrypts plaintext using AES-256-GCM. The random nonce is prepended
// to the ciphertext in the returned slice so that a single byte slice can be stored.
func aesgcmEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// aesgcmDecrypt decrypts a ciphertext produced by [aesgcmEncrypt] (nonce is prepended).
func aesgcmDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, fmt.Errorf("ciphertext too short (got %d bytes, need at least %d)", len(ciphertext), ns)
	}
	return gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
}
