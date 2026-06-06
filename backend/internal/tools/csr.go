package tools

import (
	"crypto/x509"
	"encoding/json"
)

type pemInput struct {
	Input  string `json:"input"`
	Format string `json:"format"` // optional override (unused for now; auto)
}

func handleCSR(raw json.RawMessage) (any, error) {
	var in pemInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	der, format, err := DecodeToDER(in.Input, "CERTIFICATE REQUEST")
	if err != nil {
		return nil, err
	}
	csr, err := x509.ParseCertificateRequest(der)
	if err != nil {
		return nil, err
	}
	if err := csr.CheckSignature(); err != nil {
		// Not fatal for decoding; report it.
	}

	out := map[string]any{
		"detectedFormat":     format,
		"subject":            pkixNameToMap(csr.Subject),
		"publicKey":          publicKeyInfo(csr.PublicKey),
		"signatureAlgorithm": csr.SignatureAlgorithm.String(),
		"version":            csr.Version,
		"signatureValid":     csr.CheckSignature() == nil,
	}
	if sans := sansToMap(csr.DNSNames, csr.EmailAddresses, csr.IPAddresses, csr.URIs); len(sans) > 0 {
		out["subjectAlternativeNames"] = sans
	}
	return out, nil
}
