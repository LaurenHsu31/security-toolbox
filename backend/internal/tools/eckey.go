package tools

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type ecKeyInput struct {
	Input string `json:"input"`
	Curve string `json:"curve"` // for raw point input
}

func handleECKey(raw json.RawMessage) (any, error) {
	var in ecKeyInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	input := strings.TrimSpace(in.Input)

	var curve elliptic.Curve
	var x, y *big.Int

	if strings.Contains(input, "-----BEGIN") {
		key, err := parsePKIXFromPEM(input)
		if err != nil {
			return nil, err
		}
		pub, ok := key.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("PEM is not an EC public key")
		}
		curve, x, y = pub.Curve, pub.X, pub.Y
	} else {
		c, err := curveFromName(orDefault(in.Curve, "P-256"))
		if err != nil {
			return nil, err
		}
		curve = c
		b, err := hex.DecodeString(wsRE.ReplaceAllString(input, ""))
		if err != nil {
			return nil, fmt.Errorf("point must be hex or PEM: %w", err)
		}
		if len(b) == 0 {
			return nil, errors.New("empty point")
		}
		switch b[0] {
		case 0x04:
			x, y = elliptic.Unmarshal(curve, b)
		case 0x02, 0x03:
			x, y = elliptic.UnmarshalCompressed(curve, b)
		default:
			return nil, errors.New("point must start with 04 (uncompressed) or 02/03 (compressed)")
		}
		if x == nil {
			return nil, errors.New("invalid point for the given curve")
		}
	}

	size := (curve.Params().BitSize + 7) / 8
	xb := padLeft(x.Bytes(), size)
	yb := padLeft(y.Bytes(), size)
	prefix := byte(0x02)
	if y.Bit(0) == 1 {
		prefix = 0x03
	}
	uncompressed := append([]byte{0x04}, append(xb, yb...)...)
	compressed := append([]byte{prefix}, xb...)

	return map[string]any{
		"curve":             curve.Params().Name,
		"fieldSizeBytes":    size,
		"x":                 "0x" + x.Text(16),
		"y":                 "0x" + y.Text(16),
		"uncompressedPoint": hex.EncodeToString(uncompressed),
		"compressedPoint":   hex.EncodeToString(compressed),
		"onCurve":           curve.IsOnCurve(x, y),
	}, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
