package tools

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
)

type jwkInput struct {
	Input     string `json:"input"`
	Direction string `json:"direction"` // pem-to-jwk | jwk-to-pem
}

func handleJWK(raw json.RawMessage) (any, error) {
	var in jwkInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	switch in.Direction {
	case "jwk-to-pem":
		return jwkToPEM(in.Input)
	default:
		return pemToJWK(in.Input)
	}
}

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func pemToJWK(pemStr string) (any, error) {
	key, err := parsePKIXFromPEM(pemStr)
	if err != nil {
		return nil, err
	}
	switch k := key.(type) {
	case *rsa.PublicKey:
		eBytes := big.NewInt(int64(k.E)).Bytes()
		return map[string]any{"jwk": map[string]any{
			"kty": "RSA",
			"n":   b64u(k.N.Bytes()),
			"e":   b64u(eBytes),
		}}, nil
	case *ecdsa.PublicKey:
		size := (k.Curve.Params().BitSize + 7) / 8
		return map[string]any{"jwk": map[string]any{
			"kty": "EC",
			"crv": k.Curve.Params().Name,
			"x":   b64u(padLeft(k.X.Bytes(), size)),
			"y":   b64u(padLeft(k.Y.Bytes(), size)),
		}}, nil
	default:
		return nil, fmt.Errorf("unsupported key type %T", key)
	}
}

type jwkBody struct {
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func jwkToPEM(s string) (any, error) {
	var j jwkBody
	if err := json.Unmarshal([]byte(s), &j); err != nil {
		return nil, fmt.Errorf("invalid JWK JSON: %w", err)
	}
	var pub any
	switch j.Kty {
	case "RSA":
		n, err := decodeB64UToInt(j.N)
		if err != nil {
			return nil, err
		}
		e, err := decodeB64UToInt(j.E)
		if err != nil {
			return nil, err
		}
		pub = &rsa.PublicKey{N: n, E: int(e.Int64())}
	case "EC":
		curve, err := curveFromName(j.Crv)
		if err != nil {
			return nil, err
		}
		x, err := decodeB64UToInt(j.X)
		if err != nil {
			return nil, err
		}
		y, err := decodeB64UToInt(j.Y)
		if err != nil {
			return nil, err
		}
		pub = &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	default:
		return nil, fmt.Errorf("unsupported kty %q", j.Kty)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	out := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return map[string]any{"pem": string(out)}, nil
}

func decodeB64UToInt(s string) (*big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		// tolerate padded input
		b, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			return nil, errors.New("field is not valid base64url")
		}
	}
	return new(big.Int).SetBytes(b), nil
}

func curveFromName(name string) (elliptic.Curve, error) {
	switch name {
	case "P-256":
		return elliptic.P256(), nil
	case "P-384":
		return elliptic.P384(), nil
	case "P-521":
		return elliptic.P521(), nil
	}
	return nil, fmt.Errorf("unsupported curve %q", name)
}
