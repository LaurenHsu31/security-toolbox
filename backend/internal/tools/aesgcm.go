package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// AES-GCM (NIST SP 800-38D). CCC Digital Key encrypts server payloads and
// mailbox data with id-aes128-GCM, so this exposes both decrypt and encrypt.

type aesGCMInput struct {
	Mode        string `json:"mode"` // decrypt | encrypt
	Key         string `json:"key"`
	KeyFormat   string `json:"keyFormat"`
	Nonce       string `json:"nonce"` // IV
	NonceFormat string `json:"nonceFormat"`
	Data        string `json:"data"` // ciphertext(+tag) for decrypt, plaintext for encrypt
	DataFormat  string `json:"dataFormat"`
	Tag         string `json:"tag"` // optional separate tag for decrypt
	TagFormat   string `json:"tagFormat"`
	AAD         string `json:"aad"`
	AADFormat   string `json:"aadFormat"`
	TagLen      int    `json:"tagLen"` // tag size in bytes (default 16)
}

func newGCM(key, nonce []byte, tagLen int) (cipher.AEAD, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, errors.New("AES key must be 16, 24 or 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if tagLen == 0 {
		tagLen = 16
	}
	if tagLen < 12 || tagLen > 16 {
		return nil, errors.New("GCM tag length must be 12..16 bytes")
	}
	if len(nonce) == 0 {
		return nil, errors.New("nonce (IV) is required")
	}
	// Standard NewGCM fixes a 12-byte nonce and 16-byte tag; the stdlib has a
	// constructor for a custom nonce OR a custom tag, but not both at once.
	switch {
	case len(nonce) == 12 && tagLen == 16:
		return cipher.NewGCM(block)
	case tagLen == 16:
		return cipher.NewGCMWithNonceSize(block, len(nonce))
	case len(nonce) == 12:
		return cipher.NewGCMWithTagSize(block, tagLen)
	default:
		return nil, errors.New("a non-12-byte nonce together with a non-16-byte tag is not supported")
	}
}

func handleAESGCM(raw json.RawMessage) (any, error) {
	var in aesGCMInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	key, _, err := decodeFlexibleBytes(in.Key, orDefault(in.KeyFormat, "auto"))
	if err != nil {
		return nil, fmt.Errorf("key: %w", err)
	}
	nonce, _, err := decodeFlexibleBytes(in.Nonce, orDefault(in.NonceFormat, "auto"))
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	var aad []byte
	if strings.TrimSpace(in.AAD) != "" {
		aad, _, err = decodeFlexibleBytes(in.AAD, orDefault(in.AADFormat, "auto"))
		if err != nil {
			return nil, fmt.Errorf("aad: %w", err)
		}
	}
	gcm, err := newGCM(key, nonce, in.TagLen)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(in.Mode) == "encrypt" {
		pt, _, err := decodeFlexibleBytes(in.Data, orDefault(in.DataFormat, "auto"))
		if err != nil {
			return nil, fmt.Errorf("plaintext: %w", err)
		}
		out := gcm.Seal(nil, nonce, pt, aad)
		ct, tag := out[:len(out)-gcm.Overhead()], out[len(out)-gcm.Overhead():]
		return map[string]any{
			"mode":                "encrypt",
			"keyBits":             len(key) * 8,
			"ciphertextHex":       hex.EncodeToString(ct),
			"tagHex":              hex.EncodeToString(tag),
			"ciphertextTagHex":    hex.EncodeToString(out),
			"ciphertextTagBase64": base64.StdEncoding.EncodeToString(out),
		}, nil
	}

	// decrypt
	ct, _, err := decodeFlexibleBytes(in.Data, orDefault(in.DataFormat, "auto"))
	if err != nil {
		return nil, fmt.Errorf("ciphertext: %w", err)
	}
	if strings.TrimSpace(in.Tag) != "" {
		tag, _, err := decodeFlexibleBytes(in.Tag, orDefault(in.TagFormat, "auto"))
		if err != nil {
			return nil, fmt.Errorf("tag: %w", err)
		}
		ct = append(append([]byte{}, ct...), tag...)
	}
	if len(ct) < gcm.Overhead() {
		return nil, fmt.Errorf("ciphertext shorter than the %d-byte tag", gcm.Overhead())
	}
	pt, err := gcm.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, errors.New("authentication failed: wrong key, nonce, AAD or tag")
	}
	out := map[string]any{
		"mode":            "decrypt",
		"keyBits":         len(key) * 8,
		"authenticated":   true,
		"plaintextHex":    hex.EncodeToString(pt),
		"plaintextBase64": base64.StdEncoding.EncodeToString(pt),
	}
	if utf8.Valid(pt) {
		out["plaintextUtf8"] = string(pt)
	}
	return out, nil
}
