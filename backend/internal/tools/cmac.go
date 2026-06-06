package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// AES-CMAC (NIST SP 800-38B / RFC 4493). CCC Digital Key uses CMAC-AES-128 as
// its PRF and as the C-MAC over secure-channel APDUs, so this is a core tool.

type cmacInput struct {
	Key       string `json:"key"`
	KeyFormat string `json:"keyFormat"` // hex|base64|auto
	Input     string `json:"input"`     // message
	From      string `json:"from"`      // utf8|hex|base64|auto
	TagLen    int    `json:"tagLen"`    // optional truncation, in bytes (1..16)
}

// rbConst is the SP 800-38B constant for a 128-bit block.
const rbConst = 0x87

// genSubkey derives a CMAC subkey from a block: (L<<1), XOR Rb when MSB(L)=1.
func genSubkey(in []byte) []byte {
	out := make([]byte, len(in))
	var carry byte
	for i := len(in) - 1; i >= 0; i-- {
		out[i] = in[i]<<1 | carry
		carry = in[i] >> 7
	}
	if in[0]&0x80 != 0 {
		out[len(out)-1] ^= rbConst
	}
	return out
}

func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// aesCMAC computes the full block-size CMAC tag of msg under block.
func aesCMAC(block cipher.Block, msg []byte) []byte {
	bs := block.BlockSize()

	zero := make([]byte, bs)
	l := make([]byte, bs)
	block.Encrypt(l, zero)
	k1 := genSubkey(l)
	k2 := genSubkey(k1)

	complete := len(msg) > 0 && len(msg)%bs == 0
	n := (len(msg) + bs - 1) / bs
	if n == 0 {
		n = 1
	}

	var last []byte
	if complete {
		last = xorBytes(msg[(n-1)*bs:], k1)
	} else {
		rem := msg[(n-1)*bs:]
		padded := make([]byte, bs)
		copy(padded, rem)
		padded[len(rem)] = 0x80
		last = xorBytes(padded, k2)
	}

	x := make([]byte, bs)
	for i := 0; i < n-1; i++ {
		block.Encrypt(x, xorBytes(x, msg[i*bs:(i+1)*bs]))
	}
	tag := make([]byte, bs)
	block.Encrypt(tag, xorBytes(x, last))
	return tag
}

func handleCMAC(raw json.RawMessage) (any, error) {
	var in cmacInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	key, _, err := decodeFlexibleBytes(in.Key, orDefault(in.KeyFormat, "auto"))
	if err != nil {
		return nil, fmt.Errorf("key: %w", err)
	}
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, errors.New("AES key must be 16, 24 or 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	msg, detected, err := decodeFlexibleBytes(in.Input, orDefault(in.From, "auto"))
	if err != nil {
		return nil, fmt.Errorf("message: %w", err)
	}

	tag := aesCMAC(block, msg)
	out := map[string]any{
		"algorithm":          fmt.Sprintf("AES-%d-CMAC", len(key)*8),
		"messageInterpreted": detected,
		"messageLength":      len(msg),
		"tagHex":             hex.EncodeToString(tag),
		"tagBase64":          base64.StdEncoding.EncodeToString(tag),
	}
	if in.TagLen > 0 && in.TagLen < len(tag) {
		out["truncatedTagHex"] = hex.EncodeToString(tag[:in.TagLen])
	}
	return out, nil
}
