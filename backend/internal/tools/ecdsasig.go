package tools

import (
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

type ecdsaSigInput struct {
	Input       string `json:"input"`
	From        string `json:"from"`        // der | raw
	Curve       string `json:"curve"`       // P-256 | P-384 | P-521
	InputFormat string `json:"inputFormat"` // hex | base64 | auto
}

type ecdsaASN1 struct {
	R *big.Int
	S *big.Int
}

func curveByteLen(curve string) (int, error) {
	switch curve {
	case "", "P-256", "p256", "secp256r1", "prime256v1":
		return 32, nil
	case "P-384", "p384", "secp384r1":
		return 48, nil
	case "P-521", "p521", "secp521r1":
		return 66, nil
	}
	return 0, fmt.Errorf("unknown curve %q (use P-256, P-384 or P-521)", curve)
}

func handleECDSASig(raw json.RawMessage) (any, error) {
	var in ecdsaSigInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	b, detected, err := decodeFlexibleBytes(in.Input, in.InputFormat)
	if err != nil {
		return nil, err
	}
	byteLen, err := curveByteLen(in.Curve)
	if err != nil {
		return nil, err
	}

	switch in.From {
	case "raw":
		if len(b)%2 != 0 {
			return nil, errors.New("raw signature length must be even (r||s)")
		}
		half := len(b) / 2
		r := new(big.Int).SetBytes(b[:half])
		s := new(big.Int).SetBytes(b[half:])
		der, err := asn1.Marshal(ecdsaASN1{R: r, S: s})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"inputFormat": detected,
			"direction":   "raw -> DER",
			"r":           "0x" + r.Text(16),
			"s":           "0x" + s.Text(16),
			"derHex":      hex.EncodeToString(der),
			"derBase64":   base64.StdEncoding.EncodeToString(der),
		}, nil

	default: // der
		var sig ecdsaASN1
		if _, err := asn1.Unmarshal(b, &sig); err != nil {
			return nil, fmt.Errorf("not a valid ASN.1 ECDSA signature: %w", err)
		}
		rb := padLeft(sig.R.Bytes(), byteLen)
		sb := padLeft(sig.S.Bytes(), byteLen)
		rawSig := append(append([]byte{}, rb...), sb...)
		return map[string]any{
			"inputFormat":  detected,
			"direction":    "DER -> raw",
			"curve":        fmt.Sprintf("%d-byte halves", byteLen),
			"r":            "0x" + sig.R.Text(16),
			"s":            "0x" + sig.S.Text(16),
			"rawHex":       hex.EncodeToString(rawSig),
			"rawBase64Url": base64.RawURLEncoding.EncodeToString(rawSig),
		}, nil
	}
}
