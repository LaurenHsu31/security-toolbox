package tools

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"strings"
	"time"
)

func handleCert(raw json.RawMessage) (any, error) {
	var in pemInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	der, format, err := DecodeToDER(in.Input, "CERTIFICATE")
	if err != nil {
		return nil, err
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	sha1sum := sha1.Sum(der)
	sha256sum := sha256.Sum256(der)
	now := time.Now()

	out := map[string]any{
		"detectedFormat":     format,
		"version":            c.Version,
		"serialNumber":       "0x" + strings.ToUpper(c.SerialNumber.Text(16)),
		"subject":            pkixNameToMap(c.Subject),
		"issuer":             pkixNameToMap(c.Issuer),
		"signatureAlgorithm": c.SignatureAlgorithm.String(),
		"publicKey":          publicKeyInfo(c.PublicKey),
		"validity": map[string]any{
			"notBefore":     c.NotBefore.UTC().Format(time.RFC3339),
			"notAfter":      c.NotAfter.UTC().Format(time.RFC3339),
			"expired":       now.After(c.NotAfter),
			"notYetValid":   now.Before(c.NotBefore),
			"daysRemaining": int(time.Until(c.NotAfter).Hours() / 24),
		},
		"fingerprints": map[string]any{
			"sha1":   colonHex(sha1sum[:]),
			"sha256": colonHex(sha256sum[:]),
		},
		"basicConstraints": map[string]any{
			"isCA": c.IsCA,
		},
		"keyUsage":         keyUsageNames(c.KeyUsage),
		"extendedKeyUsage": extKeyUsageNames(c.ExtKeyUsage),
		"selfSigned":       c.CheckSignatureFrom(c) == nil,
	}
	if len(c.SubjectKeyId) > 0 {
		out["subjectKeyId"] = colonHex(c.SubjectKeyId)
	}
	if len(c.AuthorityKeyId) > 0 {
		out["authorityKeyId"] = colonHex(c.AuthorityKeyId)
	}
	if sans := sansToMap(c.DNSNames, c.EmailAddresses, c.IPAddresses, c.URIs); len(sans) > 0 {
		out["subjectAlternativeNames"] = sans
	}
	if len(c.CRLDistributionPoints) > 0 {
		out["crlDistributionPoints"] = c.CRLDistributionPoints
	}
	if len(c.OCSPServer) > 0 {
		out["ocspServers"] = c.OCSPServer
	}

	// Full extension list (Certificate Detail Information): every extension by
	// OID with its critical flag and raw value, and a decoded form for the ones
	// we understand. This guarantees nothing is hidden even when Go's parser does
	// not surface an extension into a struct field.
	if exts := certExtensions(c); len(exts) > 0 {
		out["extensions"] = exts
	}

	// Raw encodings: the DER hex (starts with 30 82 …), a re-wrapped PEM, the
	// signature bytes, and the full ASN.1 tree of the certificate.
	encodings := map[string]any{
		"hexEncoded": hex.EncodeToString(der),
		"pem":        string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		"signature":  hex.EncodeToString(c.Signature),
	}
	if tree, terr := walkASN1(der, 0); terr == nil {
		encodings["asn1"] = tree
	}
	out["raw"] = encodings

	return out, nil
}

// certExtensions lists every X.509 extension with its OID, friendly name,
// critical flag and raw value, plus a decoded value for the well-known ones.
func certExtensions(c *x509.Certificate) []map[string]any {
	var out []map[string]any
	for _, e := range c.Extensions {
		oid := e.Id.String()
		m := map[string]any{
			"oid":      oid,
			"critical": e.Critical,
			"valueHex": hex.EncodeToString(e.Value),
		}
		if name := extensionName(oid); name != "" {
			m["name"] = name
		}
		if d := decodeExtension(oid, e.Value); d != nil {
			m["decoded"] = d
		}
		out = append(out, m)
	}
	return out
}

var certExtNames = map[string]string{
	"2.5.29.14":         "Subject Key Identifier",
	"2.5.29.15":         "Key Usage",
	"2.5.29.17":         "Subject Alternative Name",
	"2.5.29.19":         "Basic Constraints",
	"2.5.29.31":         "CRL Distribution Points",
	"2.5.29.32":         "Certificate Policies",
	"2.5.29.35":         "Authority Key Identifier",
	"2.5.29.37":         "Extended Key Usage",
	"1.3.6.1.5.5.7.1.1": "Authority Information Access",
}

func extensionName(oid string) string {
	if n, ok := certExtNames[oid]; ok {
		return n
	}
	return oidLookup(oid)
}

// decodeExtension interprets the extensions whose raw value is small and useful
// to show inline. Key Usage is decoded straight from the BIT STRING here so it
// is always visible regardless of how the standard library exposed it.
func decodeExtension(oid string, val []byte) any {
	switch oid {
	case "2.5.29.15": // Key Usage
		var bs asn1.BitString
		if _, err := asn1.Unmarshal(val, &bs); err != nil {
			return nil
		}
		bits := []string{
			"Digital Signature", "Content Commitment", "Key Encipherment",
			"Data Encipherment", "Key Agreement", "Certificate Sign",
			"CRL Sign", "Encipher Only", "Decipher Only",
		}
		var names []string
		for i, name := range bits {
			if bs.At(i) != 0 {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

func colonHex(b []byte) string {
	s := hex.EncodeToString(b)
	var sb strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			sb.WriteByte(':')
		}
		sb.WriteString(strings.ToUpper(s[i : i+2]))
	}
	return sb.String()
}

func keyUsageNames(u x509.KeyUsage) []string {
	var n []string
	add := func(bit x509.KeyUsage, name string) {
		if u&bit != 0 {
			n = append(n, name)
		}
	}
	add(x509.KeyUsageDigitalSignature, "Digital Signature")
	add(x509.KeyUsageContentCommitment, "Content Commitment")
	add(x509.KeyUsageKeyEncipherment, "Key Encipherment")
	add(x509.KeyUsageDataEncipherment, "Data Encipherment")
	add(x509.KeyUsageKeyAgreement, "Key Agreement")
	add(x509.KeyUsageCertSign, "Certificate Sign")
	add(x509.KeyUsageCRLSign, "CRL Sign")
	add(x509.KeyUsageEncipherOnly, "Encipher Only")
	add(x509.KeyUsageDecipherOnly, "Decipher Only")
	return n
}

func extKeyUsageNames(us []x509.ExtKeyUsage) []string {
	names := map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageAny:             "Any",
		x509.ExtKeyUsageServerAuth:      "TLS Server Authentication",
		x509.ExtKeyUsageClientAuth:      "TLS Client Authentication",
		x509.ExtKeyUsageCodeSigning:     "Code Signing",
		x509.ExtKeyUsageEmailProtection: "Email Protection",
		x509.ExtKeyUsageTimeStamping:    "Time Stamping",
		x509.ExtKeyUsageOCSPSigning:     "OCSP Signing",
	}
	var n []string
	for _, u := range us {
		if name, ok := names[u]; ok {
			n = append(n, name)
		} else {
			n = append(n, "Unknown")
		}
	}
	return n
}
