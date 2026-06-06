package tools

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"unicode/utf8"
)

type encodeInput struct {
	Input string `json:"input"`
	From  string `json:"from"` // utf8|base64|base64url|hex|auto
}

func handleEncode(raw json.RawMessage) (any, error) {
	var in encodeInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	b, detected, err := decodeFlexibleBytes(in.Input, in.From)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"detectedInput": detected,
		"byteLength":    len(b),
		"hex":           hex.EncodeToString(b),
		"base64":        base64.StdEncoding.EncodeToString(b),
		"base64url":     base64.RawURLEncoding.EncodeToString(b),
	}
	if utf8.Valid(b) {
		out["utf8"] = string(b)
	} else {
		out["utf8"] = nil
		out["note"] = "bytes are not valid UTF-8 text"
	}
	return out, nil
}
