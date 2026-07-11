package tools

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rsa"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type jwtInput struct {
	Token     string `json:"token"`
	Secret    string `json:"secret"`    // for HS* (raw string)
	PublicKey string `json:"publicKey"` // PEM, for RS*/PS*/ES*
}

func handleJWT(raw json.RawMessage) (any, error) {
	var in jwtInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	tok := strings.TrimSpace(in.Token)
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return nil, errors.New("a JWT must have 3 dot-separated parts")
	}

	header, err := decodeJWTSegment(parts[0])
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	payload, err := decodeJWTSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}

	alg, _ := header["alg"].(string)
	out := map[string]any{
		"header":  header,
		"payload": payload,
		"claims":  annotateClaims(payload),
	}

	sig := map[string]any{"algorithm": alg, "status": "not verified"}
	if in.Secret != "" || in.PublicKey != "" {
		ok, verr := verifyJWT(alg, parts[0]+"."+parts[1], parts[2], in.Secret, in.PublicKey)
		switch {
		case verr != nil:
			sig["status"] = "error"
			sig["error"] = verr.Error()
		case ok:
			sig["status"] = "valid"
		default:
			sig["status"] = "invalid"
		}
	}
	out["signature"] = sig
	return out, nil
}

// b64uDecode decodes base64url, tolerating the padding some encoders emit
// even though RFC 7515 says segments are unpadded.
func b64uDecode(seg string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(seg, "="))
}

func decodeJWTSegment(seg string) (map[string]any, error) {
	b, err := b64uDecode(seg)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func annotateClaims(p map[string]any) map[string]any {
	out := map[string]any{}
	for _, c := range []string{"exp", "iat", "nbf"} {
		if v, ok := p[c]; ok {
			if f, ok := v.(float64); ok {
				t := time.Unix(int64(f), 0).UTC()
				out[c] = map[string]any{
					"epoch":   int64(f),
					"utc":     t.Format(time.RFC3339),
					"expired": c == "exp" && time.Now().After(t),
				}
			}
		}
	}
	for _, c := range []string{"iss", "sub", "aud", "jti"} {
		if v, ok := p[c]; ok {
			out[c] = v
		}
	}
	return out
}

func verifyJWT(alg, signingInput, sigB64, secret, pubPEM string) (bool, error) {
	sig, err := b64uDecode(sigB64)
	if err != nil {
		return false, fmt.Errorf("signature is not valid base64url: %w", err)
	}
	h, err := hashForAlg(alg)
	if err != nil {
		return false, err
	}
	digest := func() []byte {
		hh := h.New()
		hh.Write([]byte(signingInput))
		return hh.Sum(nil)
	}

	switch {
	case strings.HasPrefix(alg, "HS"):
		if secret == "" {
			return false, errors.New("HS* needs a secret")
		}
		mac := hmac.New(h.New, []byte(secret))
		mac.Write([]byte(signingInput))
		return hmac.Equal(mac.Sum(nil), sig), nil

	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "PS"):
		pub, err := parseRSAPublic(pubPEM)
		if err != nil {
			return false, err
		}
		if strings.HasPrefix(alg, "PS") {
			// Auto-detect the salt length: RFC 7518 mandates salt = hash length,
			// but verification should accept any salt the signer used.
			return rsa.VerifyPSS(pub, h, digest(), sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto, Hash: h}) == nil, nil
		}
		return rsa.VerifyPKCS1v15(pub, h, digest(), sig) == nil, nil

	case strings.HasPrefix(alg, "ES"):
		pub, err := parseECPublic(pubPEM)
		if err != nil {
			return false, err
		}
		if len(sig)%2 != 0 {
			return false, errors.New("ECDSA signature length is odd")
		}
		half := len(sig) / 2
		r := new(big.Int).SetBytes(sig[:half])
		s := new(big.Int).SetBytes(sig[half:])
		return ecdsa.Verify(pub, digest(), r, s), nil
	}
	return false, fmt.Errorf("unsupported alg %q", alg)
}

func hashForAlg(alg string) (crypto.Hash, error) {
	switch {
	case strings.HasSuffix(alg, "256"):
		return crypto.SHA256, nil
	case strings.HasSuffix(alg, "384"):
		return crypto.SHA384, nil
	case strings.HasSuffix(alg, "512"):
		return crypto.SHA512, nil
	}
	return 0, fmt.Errorf("cannot determine hash for alg %q", alg)
}

func parsePKIXFromPEM(pemStr string) (any, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemStr)))
	if block == nil {
		return nil, errors.New("public key is not valid PEM")
	}
	if block.Type == "CERTIFICATE" {
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		return c.PublicKey, nil
	}
	return x509.ParsePKIXPublicKey(block.Bytes)
}

func parseRSAPublic(pemStr string) (*rsa.PublicKey, error) {
	k, err := parsePKIXFromPEM(pemStr)
	if err != nil {
		return nil, err
	}
	pub, ok := k.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("PEM is not an RSA public key")
	}
	return pub, nil
}

func parseECPublic(pemStr string) (*ecdsa.PublicKey, error) {
	k, err := parsePKIXFromPEM(pemStr)
	if err != nil {
		return nil, err
	}
	pub, ok := k.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("PEM is not an EC public key")
	}
	return pub, nil
}
