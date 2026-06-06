package tools

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrUnknownTool is returned by Run when the tool name is not registered.
var ErrUnknownTool = errors.New("unknown tool")

// Handler runs one conversion. raw is the request JSON body.
type Handler func(raw json.RawMessage) (any, error)

// Tool is the public metadata the frontend uses to render the UI.
type Tool struct {
	Name        string  `json:"name"`
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Handler     Handler `json:"-"`
}

// ordered list -> stable UI ordering
var registry = []Tool{
	{"csr", "CSR Decoder", "Certificates", "Decode a PKCS#10 certificate signing request. Auto-detects PEM / DER / Base64.", handleCSR},
	{"cert", "X.509 Certificate", "Certificates", "Decode an X.509 certificate: subject, issuer, validity, extensions, fingerprints.", handleCert},
	{"pkcs7", "PKCS#7 / CMS", "Certificates", "Inspect the ASN.1 structure of a PKCS#7 / CMS container.", handlePKCS7},
	{"jwt", "JWT", "Tokens", "Decode and (optionally) verify a JSON Web Token. Shows claims with human-readable times.", handleJWT},
	{"json", "JSON Formatter", "Data", "Validate, beautify or minify JSON, with precise error position.", handleJSON},
	{"cbor", "CBOR Decoder", "Data", "Decode CBOR (RFC 8949) into readable JSON-like output.", handleCBOR},
	{"cose", "COSE Decoder", "Crypto", "Decode COSE_Sign1 / COSE_Key structures used in attestation.", handleCOSE},
	{"ecdsa-sig", "ECDSA Signature", "Crypto", "Convert an ECDSA signature between ASN.1 DER and raw r||s (JWS/ES256) form.", handleECDSASig},
	{"cmac", "AES-CMAC", "Crypto", "Compute an AES-CMAC tag (NIST SP800-38B / RFC 4493). CCC uses CMAC-AES-128.", handleCMAC},
	{"hkdf", "HKDF", "Crypto", "Derive key material with HKDF (RFC 5869) extract-and-expand. Defaults to SHA-256.", handleHKDF},
	{"aes-gcm", "AES-GCM", "Crypto", "Encrypt or decrypt with AES-GCM (NIST SP800-38D), as used for CCC server payloads.", handleAESGCM},
	{"ecdh", "ECDH", "Crypto", "Compute an ECDH shared secret from a private key and a peer public key.", handleECDH},
	{"jwk", "JWK \u2194 PEM", "Crypto", "Convert public keys between JWK and PEM (RSA and EC).", handleJWK},
	{"eckey", "EC Key Inspector", "Crypto", "Inspect an EC public key/point: curve, coordinates, compressed/uncompressed.", handleECKey},
	{"hash", "Hash", "Crypto", "Compute MD5 / SHA-1 / SHA-256 / SHA-384 / SHA-512 of text, hex or base64 input.", handleHash},
	{"asn1", "ASN.1 / DER Dump", "Encoding", "Recursively decode any DER/BER structure into a readable tree.", handleASN1},
	{"encode", "Base64 / Hex", "Encoding", "Convert freely between UTF-8 text, Base64, Base64URL and Hex.", handleEncode},
	{"oid", "OID Lookup", "Encoding", "Look up an object identifier by dotted notation or name.", handleOID},
	{"tlv", "TLV / APDU", "Smartcard", "Parse BER-TLV data and ISO 7816-4 command/response APDUs.", handleTLV},
}

var byName = func() map[string]Tool {
	m := make(map[string]Tool, len(registry))
	for _, t := range registry {
		m[t.Name] = t
	}
	return m
}()

// All returns tool metadata for the frontend.
func All() []Tool { return registry }

// Run dispatches to a tool handler.
func Run(name string, raw json.RawMessage) (any, error) {
	t, ok := byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
	return t.Handler(raw)
}
