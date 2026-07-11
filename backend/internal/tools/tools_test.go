package tools

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// run invokes a handler with the given request body and asserts it returned a
// map without error. (Go forbids passing a two-value call as one argument
// among others, so the handler is invoked here rather than inline.)
func run(t *testing.T, h Handler, body any) map[string]any {
	t.Helper()
	out, err := h(mustJSON(t, body))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	m, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", out)
	}
	return m
}

func TestCSRDecode(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "vehicle.example", Organization: []string{"CCC"}},
		DNSNames: []string{"vehicle.example"},
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tmpl, key)
	if err != nil {
		t.Fatal(err)
	}
	p := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))

	out := run(t, handleCSR, map[string]any{"input": p})
	subj := out["subject"].(map[string]any)
	if subj["commonName"] != "vehicle.example" {
		t.Errorf("CN = %v", subj["commonName"])
	}
	if pk := out["publicKey"].(map[string]any); pk["algorithm"] != "ECDSA" {
		t.Errorf("alg = %v", pk["algorithm"])
	}
	if out["signatureValid"] != true {
		t.Errorf("signature should be valid")
	}
}

func TestCSRMangledMarkers(t *testing.T) {
	// A self-generated CSR whose BEGIN/END markers are then corrupted the way a
	// copy-paste from a PDF/word processor does: U+2011 non-breaking hyphens
	// instead of ASCII '-', and only four trailing dashes. It should still decode.
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: "Example Vehicle CA", Organization: []string{"Example Org"}},
	}, key)
	if err != nil {
		t.Fatal(err)
	}
	clean := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))

	const nbh = "‑" // U+2011 non-breaking hyphen, not ASCII '-'
	mangled := strings.NewReplacer(
		"-----BEGIN CERTIFICATE REQUEST-----",
		strings.Repeat(nbh, 5)+"BEGIN CERTIFICATE REQUEST"+strings.Repeat(nbh, 4),
		"-----END CERTIFICATE REQUEST-----",
		strings.Repeat(nbh, 5)+"END CERTIFICATE REQUEST"+strings.Repeat(nbh, 4),
	).Replace(clean)

	out := run(t, handleCSR, map[string]any{"input": mangled})
	if cn := out["subject"].(map[string]any)["commonName"]; cn != "Example Vehicle CA" {
		t.Errorf("CN = %v", cn)
	}
}

func TestCertEscapedAndSingleLine(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(7),
		Subject:      pkix.Name{CommonName: "oneline.example"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	clean := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))

	cases := map[string]string{
		"literal \\n escapes": strings.ReplaceAll(clean, "\n", `\n`),
		"all on one line":     strings.ReplaceAll(clean, "\n", ""),
		"literal \\r\\n":      strings.ReplaceAll(clean, "\n", `\r\n`),
	}
	for name, input := range cases {
		out := run(t, handleCert, map[string]any{"input": input})
		if cn := out["subject"].(map[string]any)["commonName"]; cn != "oneline.example" {
			t.Errorf("%s: CN = %v", name, cn)
		}
	}
}

func TestCertDecode(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(42),
		Subject:               pkix.Name{CommonName: "root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	p := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	out := run(t, handleCert, map[string]any{"input": p})
	if bc := out["basicConstraints"].(map[string]any); bc["isCA"] != true {
		t.Errorf("isCA should be true")
	}
	ku := out["keyUsage"].([]string)
	if len(ku) == 0 {
		t.Errorf("expected key usage entries")
	}

	// The full extension list must include Key Usage decoded from its raw
	// extension, plus the DER hex / ASN.1 raw sections.
	exts := out["extensions"].([]map[string]any)
	var sawKeyUsage bool
	for _, e := range exts {
		if e["oid"] == "2.5.29.15" {
			sawKeyUsage = true
			if dec, ok := e["decoded"].([]string); !ok || len(dec) == 0 {
				t.Errorf("Key Usage extension not decoded: %v", e["decoded"])
			}
		}
	}
	if !sawKeyUsage {
		t.Errorf("Key Usage extension (2.5.29.15) missing from extensions list")
	}
	raw := out["raw"].(map[string]any)
	if hexEnc, _ := raw["hexEncoded"].(string); !strings.HasPrefix(hexEnc, "3082") {
		t.Errorf("DER hex should start with 3082, got %.8s", hexEnc)
	}
	if _, ok := raw["asn1"]; !ok {
		t.Errorf("expected ASN.1 tree in raw section")
	}
}

func TestJWTHmac(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"123","exp":4102444800}`))
	signing := header + "." + payload
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(signing))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := signing + "." + sig

	out := run(t, handleJWT, map[string]any{"token": token, "secret": "secret"})
	s := out["signature"].(map[string]any)
	if s["status"] != "valid" {
		t.Errorf("expected valid, got %v", s["status"])
	}
	if out["payload"].(map[string]any)["sub"] != "123" {
		t.Errorf("payload sub mismatch")
	}
}

func TestJSONBeautify(t *testing.T) {
	out := run(t, handleJSON, map[string]any{"input": `{"a":1,"b":[2,3]}`, "mode": "beautify", "indent": 2})
	if out["valid"] != true {
		t.Fatal("should be valid")
	}
	if _, ok := out["formatted"].(string); !ok {
		t.Fatal("expected formatted string")
	}
	// beautify also returns the parsed value for the collapsible tree view.
	parsed, ok := out["parsed"].(map[string]any)
	if !ok {
		t.Fatalf("expected parsed object, got %T", out["parsed"])
	}
	if _, ok := parsed["a"]; !ok {
		t.Errorf("parsed should contain key a: %v", parsed)
	}
	if _, err := handleJSON(mustJSON(t, map[string]any{"input": `{"a":}`})); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestECDSASigRoundTrip(t *testing.T) {
	r, _ := new(big.Int).SetString("123456789abcdef0", 16)
	s, _ := new(big.Int).SetString("fedcba9876543210", 16)
	der, _ := asn1.Marshal(ecdsaASN1{R: r, S: s})

	fromDER := run(t, handleECDSASig, map[string]any{
		"input": hex.EncodeToString(der), "from": "der", "curve": "P-256", "inputFormat": "hex",
	})
	rawHex := fromDER["rawHex"].(string)
	if len(rawHex) != 128 { // 64 bytes
		t.Errorf("raw P-256 sig should be 64 bytes, got %d hex chars", len(rawHex))
	}

	back := run(t, handleECDSASig, map[string]any{
		"input": rawHex, "from": "raw", "curve": "P-256", "inputFormat": "hex",
	})
	if back["derHex"].(string) != hex.EncodeToString(der) {
		t.Errorf("round trip mismatch:\n got %v\nwant %v", back["derHex"], hex.EncodeToString(der))
	}
}

func TestEncode(t *testing.T) {
	out := run(t, handleEncode, map[string]any{"input": "hello", "from": "utf8"})
	if out["base64"] != "aGVsbG8=" {
		t.Errorf("base64 = %v", out["base64"])
	}
	if out["hex"] != "68656c6c6f" {
		t.Errorf("hex = %v", out["hex"])
	}
}

func TestHash(t *testing.T) {
	out := run(t, handleHash, map[string]any{"input": "abc", "from": "utf8"})
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if out["sha256"] != want {
		t.Errorf("sha256(abc) = %v", out["sha256"])
	}
}

func TestTLVAPDUCommand(t *testing.T) {
	out := run(t, handleTLV, map[string]any{
		"input": "00A4040007A0000002471001", "mode": "apdu-command",
	})
	if out["insName"] != "SELECT" {
		t.Errorf("ins = %v", out["insName"])
	}
	if out["lc"].(int) != 7 {
		t.Errorf("lc = %v", out["lc"])
	}
}

func TestJWKRoundTrip(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	p := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	toJWK := run(t, handleJWK, map[string]any{"input": p, "direction": "pem-to-jwk"})
	jwk := toJWK["jwk"].(map[string]any)
	if jwk["kty"] != "RSA" {
		t.Fatalf("kty = %v", jwk["kty"])
	}
	jwkJSON, _ := json.Marshal(jwk)
	back := run(t, handleJWK, map[string]any{"input": string(jwkJSON), "direction": "jwk-to-pem"})
	block, _ := pem.Decode([]byte(back["pem"].(string)))
	if block == nil {
		t.Fatal("output is not PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if pub.(*rsa.PublicKey).N.Cmp(key.PublicKey.N) != 0 {
		t.Error("modulus mismatch after round trip")
	}
}

func TestCBOR(t *testing.T) {
	// {"a":1, "b":[2,3]} in CBOR: a2 61 61 01 61 62 82 02 03
	out := run(t, handleCBOR, map[string]any{"input": "a26161016162820203", "inputFormat": "hex"})
	m := out["decoded"].(map[string]any)
	if _, ok := m["a"]; !ok {
		t.Errorf("expected key a in %v", m)
	}
}

func TestCMAC(t *testing.T) {
	// RFC 4493 test vectors, key = 2b7e1516...
	key := "2b7e151628aed2a6abf7158809cf4f3c"
	empty := run(t, handleCMAC, map[string]any{"key": key, "input": "", "from": "hex"})
	if empty["tagHex"] != "bb1d6929e95937287fa37d129b756746" {
		t.Errorf("CMAC of empty = %v", empty["tagHex"])
	}
	one := run(t, handleCMAC, map[string]any{
		"key": key, "input": "6bc1bee22e409f96e93d7e117393172a", "from": "hex",
	})
	if one["tagHex"] != "070a16b46b4d4144f79bdd9dd04a287c" {
		t.Errorf("CMAC of one block = %v", one["tagHex"])
	}
}

func TestHKDF(t *testing.T) {
	// RFC 5869 Test Case 1.
	out := run(t, handleHKDF, map[string]any{
		"ikm":    "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
		"salt":   "000102030405060708090a0b0c",
		"info":   "f0f1f2f3f4f5f6f7f8f9",
		"length": 42,
		"hash":   "SHA-256",
	})
	if out["prkHex"] != "077709362c2e32df0ddc3f0dc47bba6390b6c73bb50f9c3122ec844ad7c2b3e5" {
		t.Errorf("PRK = %v", out["prkHex"])
	}
	want := "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865"
	if out["okmHex"] != want {
		t.Errorf("OKM = %v", out["okmHex"])
	}
}

func TestAESGCM(t *testing.T) {
	// NIST GCM test case 2: 128-bit zero key/IV, 16-byte zero plaintext.
	zeroKey := "00000000000000000000000000000000"
	zeroIV := "000000000000000000000000"
	want := "0388dace60b6a392f328c2b971b2fe78ab6e47d42cec13bdf53a67b21257bddf"

	enc := run(t, handleAESGCM, map[string]any{
		"mode": "encrypt", "key": zeroKey, "nonce": zeroIV, "data": zeroKey,
	})
	if enc["ciphertextTagHex"] != want {
		t.Errorf("ciphertext+tag = %v", enc["ciphertextTagHex"])
	}
	dec := run(t, handleAESGCM, map[string]any{
		"mode": "decrypt", "key": zeroKey, "nonce": zeroIV, "data": want,
	})
	if dec["plaintextHex"] != zeroKey {
		t.Errorf("plaintext = %v", dec["plaintextHex"])
	}
	// Wrong key must fail authentication.
	if _, err := handleAESGCM(mustJSON(t, map[string]any{
		"mode": "decrypt", "key": "ff000000000000000000000000000000", "nonce": zeroIV, "data": want,
	})); err == nil {
		t.Error("expected authentication failure with wrong key")
	}
}

func TestECDH(t *testing.T) {
	a, _ := ecdh.P256().GenerateKey(rand.Reader)
	b, _ := ecdh.P256().GenerateKey(rand.Reader)
	out1 := run(t, handleECDH, map[string]any{
		"private": hex.EncodeToString(a.Bytes()),
		"public":  hex.EncodeToString(b.PublicKey().Bytes()),
		"curve":   "P-256",
	})
	out2 := run(t, handleECDH, map[string]any{
		"private": hex.EncodeToString(b.Bytes()),
		"public":  hex.EncodeToString(a.PublicKey().Bytes()),
		"curve":   "P-256",
	})
	if out1["sharedSecretHex"] != out2["sharedSecretHex"] {
		t.Errorf("ECDH mismatch: %v vs %v", out1["sharedSecretHex"], out2["sharedSecretHex"])
	}
}

func TestCBORIndefiniteByteString(t *testing.T) {
	// (_ h'AABB' h'CC') → 5f 42 aabb 41 cc ff: chunks must concatenate as raw
	// bytes, not as their "h'…'" display strings.
	out := run(t, handleCBOR, map[string]any{"input": "5f42aabb41ccff", "inputFormat": "hex"})
	if out["decoded"] != "h'aabbcc'" {
		t.Errorf("decoded = %v, want h'aabbcc'", out["decoded"])
	}
}

func TestCBORHugeLengthErrors(t *testing.T) {
	// mt 2 (byte string), ai 27 with length 0xFFFFFFFFFFFFFFFF must error, not
	// overflow int and panic.
	if _, err := handleCBOR(mustJSON(t, map[string]any{"input": "5bffffffffffffffff", "inputFormat": "hex"})); err == nil {
		t.Error("expected error for oversized CBOR length")
	}
}

func TestTLVBadLengths(t *testing.T) {
	if _, err := handleTLV(mustJSON(t, map[string]any{"input": "3080"})); err == nil {
		t.Error("indefinite length (0x80) should error, not parse as zero length")
	}
	if _, err := handleTLV(mustJSON(t, map[string]any{"input": "0485ffffffffff"})); err == nil {
		t.Error("5-byte length field should error, not overflow")
	}
}

func TestAPDULeZeroMeans256(t *testing.T) {
	out := run(t, handleTLV, map[string]any{"input": "00B0000000", "mode": "apdu-command"})
	if out["le"].(int) != 256 {
		t.Errorf("le = %v, want 256 for Le byte 00", out["le"])
	}
}

func TestJWTPaddedSegments(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"123"}`))
	signing := header + "." + payload
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(signing))
	// Padded signature segment (some encoders emit padding despite RFC 7515).
	sig := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	if !strings.Contains(sig, "=") {
		t.Fatal("test setup: expected padded signature")
	}
	out := run(t, handleJWT, map[string]any{"token": signing + "." + sig, "secret": "secret"})
	if s := out["signature"].(map[string]any); s["status"] != "valid" {
		t.Errorf("expected valid with padded segment, got %v (%v)", s["status"], s["error"])
	}
}

func TestJWTPSSAnySaltLength(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"PS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"x"}`))
	signing := header + "." + payload
	digest := sha256.Sum256([]byte(signing))
	// Salt length 20 ≠ hash length 32: verification must still accept it.
	sig, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: 20})
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	token := signing + "." + base64.RawURLEncoding.EncodeToString(sig)
	out := run(t, handleJWT, map[string]any{"token": token, "publicKey": pubPEM})
	if s := out["signature"].(map[string]any); s["status"] != "valid" {
		t.Errorf("expected valid PSS signature, got %v (%v)", s["status"], s["error"])
	}
}

func TestDecodeAutoFormats(t *testing.T) {
	// Unpadded standard base64 containing '+' must not fall through to UTF-8.
	out := run(t, handleEncode, map[string]any{"input": "+7w", "from": "auto"})
	if out["hex"] != "fbbc" {
		t.Errorf("hex = %v, want fbbc (detected %v)", out["hex"], out["detectedInput"])
	}
	// 0x-prefixed hex should decode as hex, in auto and explicit modes.
	out = run(t, handleEncode, map[string]any{"input": "0xDEADbeef", "from": "auto"})
	if out["hex"] != "deadbeef" {
		t.Errorf("auto 0x hex = %v", out["hex"])
	}
	out = run(t, handleEncode, map[string]any{"input": "0xdeadbeef", "from": "hex"})
	if out["hex"] != "deadbeef" {
		t.Errorf("explicit 0x hex = %v", out["hex"])
	}
	// Unpadded standard base64 in explicit base64 mode.
	out = run(t, handleEncode, map[string]any{"input": "aGVsbG8", "from": "base64"})
	if out["utf8"] != "hello" {
		t.Errorf("unpadded base64 = %v", out["utf8"])
	}
}

func TestNumericFieldsAcceptStrings(t *testing.T) {
	// The frontend sends every control value as a string ("" when untouched);
	// numeric fields must tolerate that instead of failing to unmarshal.
	out := run(t, handleHKDF, map[string]any{
		"ikm": "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
		"salt": "000102030405060708090a0b0c", "info": "f0f1f2f3f4f5f6f7f8f9",
		"length": "42", "hash": "SHA-256",
	})
	if out["length"] != 42 {
		t.Errorf("length = %v", out["length"])
	}
	out = run(t, handleAESGCM, map[string]any{
		"mode": "decrypt", "key": "00000000000000000000000000000000",
		"nonce": "000000000000000000000000",
		"data":  "0388dace60b6a392f328c2b971b2fe78ab6e47d42cec13bdf53a67b21257bddf",
		"tagLen": "", "tag": "", "aad": "",
	})
	if out["authenticated"] != true {
		t.Errorf("decrypt with string tagLen failed")
	}
	out = run(t, handleCMAC, map[string]any{
		"key": "2b7e151628aed2a6abf7158809cf4f3c", "input": "", "from": "hex", "tagLen": "8",
	})
	if out["truncatedTagHex"] != "bb1d6929e9593728" {
		t.Errorf("truncated tag = %v", out["truncatedTagHex"])
	}
	if _, err := handleHKDF(mustJSON(t, map[string]any{"ikm": "0b0b", "length": "abc"})); err == nil {
		t.Error("expected error for non-numeric length")
	}
}

func TestASN1Time(t *testing.T) {
	der, err := asn1.Marshal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	out := run(t, handleASN1, map[string]any{"input": hex.EncodeToString(der)})
	tree := out["tree"].([]map[string]any)
	v, _ := tree[0]["value"].(string)
	if !strings.Contains(v, "2030-01-02T03:04:05Z") {
		t.Errorf("UTCTime not interpreted: %v", v)
	}
}

func TestASN1(t *testing.T) {
	der, _ := asn1.Marshal(struct {
		A int
		B string `asn1:"printable"`
	}{A: 7, B: "hi"})
	out := run(t, handleASN1, map[string]any{"input": hex.EncodeToString(der)})
	tree := out["tree"].([]map[string]any)
	if len(tree) != 1 || tree[0]["tagName"] != "SEQUENCE" {
		t.Errorf("expected top-level SEQUENCE, got %v", tree)
	}
}
