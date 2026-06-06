package tools

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
)

// HKDF (RFC 5869), extract-and-expand. CCC Digital Key derives session keys
// with HKDF-SHA256. Implemented on crypto/hmac to keep the binary zero-dep
// (crypto/hkdf only landed in Go 1.24).

type hkdfInput struct {
	IKM        string `json:"ikm"`
	IKMFormat  string `json:"ikmFormat"`
	Salt       string `json:"salt"`
	SaltFormat string `json:"saltFormat"`
	Info       string `json:"info"`
	InfoFormat string `json:"infoFormat"`
	Length     int    `json:"length"`
	Hash       string `json:"hash"` // SHA-256 | SHA-384 | SHA-512
}

func hashByName(name string) (func() hash.Hash, int, error) {
	switch name {
	case "", "SHA-256", "sha256":
		return sha256.New, sha256.Size, nil
	case "SHA-384", "sha384":
		return sha512.New384, sha512.Size384, nil
	case "SHA-512", "sha512":
		return sha512.New, sha512.Size, nil
	}
	return nil, 0, fmt.Errorf("unsupported hash %q (use SHA-256, SHA-384 or SHA-512)", name)
}

func handleHKDF(raw json.RawMessage) (any, error) {
	var in hkdfInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	newHash, hashLen, err := hashByName(in.Hash)
	if err != nil {
		return nil, err
	}

	ikm, _, err := decodeFlexibleBytes(in.IKM, orDefault(in.IKMFormat, "auto"))
	if err != nil {
		return nil, fmt.Errorf("IKM: %w", err)
	}

	salt := make([]byte, hashLen) // RFC 5869: absent salt = HashLen zeros
	if strings.TrimSpace(in.Salt) != "" {
		salt, _, err = decodeFlexibleBytes(in.Salt, orDefault(in.SaltFormat, "auto"))
		if err != nil {
			return nil, fmt.Errorf("salt: %w", err)
		}
	}

	var info []byte
	if strings.TrimSpace(in.Info) != "" {
		info, _, err = decodeFlexibleBytes(in.Info, orDefault(in.InfoFormat, "auto"))
		if err != nil {
			return nil, fmt.Errorf("info: %w", err)
		}
	}

	length := in.Length
	if length <= 0 {
		length = hashLen
	}
	if length > 255*hashLen {
		return nil, fmt.Errorf("length %d too large (max %d for this hash)", length, 255*hashLen)
	}

	// Extract.
	ext := hmac.New(newHash, salt)
	ext.Write(ikm)
	prk := ext.Sum(nil)

	// Expand.
	okm := make([]byte, 0, length)
	var t []byte
	for i := 1; len(okm) < length; i++ {
		h := hmac.New(newHash, prk)
		h.Write(t)
		h.Write(info)
		h.Write([]byte{byte(i)})
		t = h.Sum(nil)
		okm = append(okm, t...)
	}
	okm = okm[:length]

	return map[string]any{
		"hash":      orDefault(in.Hash, "SHA-256"),
		"prkHex":    hex.EncodeToString(prk),
		"okmHex":    hex.EncodeToString(okm),
		"okmBase64": base64.StdEncoding.EncodeToString(okm),
		"length":    length,
	}, nil
}
