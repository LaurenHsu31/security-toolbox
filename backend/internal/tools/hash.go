package tools

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
)

type hashInput struct {
	Input string `json:"input"`
	From  string `json:"from"` // utf8|hex|base64|auto
}

func handleHash(raw json.RawMessage) (any, error) {
	var in hashInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	from := in.From
	if from == "" {
		from = "utf8" // hashing text is the common case
	}
	b, detected, err := decodeFlexibleBytes(in.Input, from)
	if err != nil {
		return nil, err
	}
	md5sum := md5.Sum(b)
	sha1sum := sha1.Sum(b)
	s256 := sha256.Sum256(b)
	s384 := sha512.Sum384(b)
	s512 := sha512.Sum512(b)
	return map[string]any{
		"inputInterpretedAs": detected,
		"byteLength":         len(b),
		"md5":                hex.EncodeToString(md5sum[:]),
		"sha1":               hex.EncodeToString(sha1sum[:]),
		"sha256":             hex.EncodeToString(s256[:]),
		"sha384":             hex.EncodeToString(s384[:]),
		"sha512":             hex.EncodeToString(s512[:]),
	}, nil
}
