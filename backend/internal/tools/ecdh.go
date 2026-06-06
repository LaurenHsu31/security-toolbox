package tools

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ECDH key agreement (crypto/ecdh, stdlib since Go 1.20). CCC Digital Key uses
// ephemeral ECDH on P-256 to establish the shared secret fed into HKDF.

type ecdhInput struct {
	Private string `json:"private"` // hex scalar or PEM (PKCS#8 / EC PRIVATE KEY)
	Public  string `json:"public"`  // hex point or PEM (SubjectPublicKeyInfo)
	Curve   string `json:"curve"`   // P-256 | P-384 | P-521 | X25519
}

func ecdhCurve(name string) (ecdh.Curve, error) {
	switch name {
	case "", "P-256", "p256", "secp256r1", "prime256v1":
		return ecdh.P256(), nil
	case "P-384", "p384", "secp384r1":
		return ecdh.P384(), nil
	case "P-521", "p521", "secp521r1":
		return ecdh.P521(), nil
	case "X25519", "x25519":
		return ecdh.X25519(), nil
	}
	return nil, fmt.Errorf("unsupported curve %q", name)
}

func parseECDHPrivate(curve ecdh.Curve, input string) (*ecdh.PrivateKey, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("private key is required")
	}
	if strings.Contains(input, "-----BEGIN") {
		der, _, err := DecodeToDER(input, "")
		if err != nil {
			return nil, err
		}
		if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
			switch p := k.(type) {
			case *ecdsa.PrivateKey:
				return p.ECDH()
			case *ecdh.PrivateKey:
				return p, nil
			}
			return nil, errors.New("PEM is not an EC private key")
		}
		ec, err := x509.ParseECPrivateKey(der)
		if err != nil {
			return nil, errors.New("could not parse private key (expected PKCS#8 or SEC1 EC)")
		}
		return ec.ECDH()
	}
	b, err := hex.DecodeString(wsRE.ReplaceAllString(input, ""))
	if err != nil {
		return nil, fmt.Errorf("private key must be a hex scalar or PEM: %w", err)
	}
	return curve.NewPrivateKey(b)
}

func parseECDHPublic(curve ecdh.Curve, input string) (*ecdh.PublicKey, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("peer public key is required")
	}
	if strings.Contains(input, "-----BEGIN") {
		der, _, err := DecodeToDER(input, "")
		if err != nil {
			return nil, err
		}
		k, err := x509.ParsePKIXPublicKey(der)
		if err != nil {
			return nil, err
		}
		switch p := k.(type) {
		case *ecdsa.PublicKey:
			return p.ECDH()
		case *ecdh.PublicKey:
			return p, nil
		}
		return nil, errors.New("PEM is not an EC public key")
	}
	b, err := hex.DecodeString(wsRE.ReplaceAllString(input, ""))
	if err != nil {
		return nil, fmt.Errorf("public key must be a hex point or PEM: %w", err)
	}
	return curve.NewPublicKey(b)
}

func handleECDH(raw json.RawMessage) (any, error) {
	var in ecdhInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	curve, err := ecdhCurve(in.Curve)
	if err != nil {
		return nil, err
	}
	priv, err := parseECDHPrivate(curve, in.Private)
	if err != nil {
		return nil, fmt.Errorf("private: %w", err)
	}
	pub, err := parseECDHPublic(curve, in.Public)
	if err != nil {
		return nil, fmt.Errorf("public: %w", err)
	}
	secret, err := priv.ECDH(pub)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed (mismatched curves?): %w", err)
	}
	return map[string]any{
		"curve":              orDefault(in.Curve, "P-256"),
		"byteLength":         len(secret),
		"sharedSecretHex":    hex.EncodeToString(secret),
		"sharedSecretBase64": base64.StdEncoding.EncodeToString(secret),
	}, nil
}
