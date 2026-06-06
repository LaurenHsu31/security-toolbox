package tools

import (
	"encoding/json"
	"regexp"
	"strings"
)

var oidMap = map[string]string{
	// Distinguished name attributes
	"2.5.4.3":              "commonName (CN)",
	"2.5.4.6":              "countryName (C)",
	"2.5.4.7":              "localityName (L)",
	"2.5.4.8":              "stateOrProvinceName (ST)",
	"2.5.4.10":             "organizationName (O)",
	"2.5.4.11":             "organizationalUnitName (OU)",
	"2.5.4.5":              "serialNumber",
	"1.2.840.113549.1.9.1": "emailAddress",
	// Public key algorithms
	"1.2.840.113549.1.1.1": "rsaEncryption",
	"1.2.840.10045.2.1":    "ecPublicKey",
	"1.3.101.112":          "Ed25519",
	// Signature algorithms
	"1.2.840.113549.1.1.11": "sha256WithRSAEncryption",
	"1.2.840.113549.1.1.12": "sha384WithRSAEncryption",
	"1.2.840.113549.1.1.13": "sha512WithRSAEncryption",
	"1.2.840.10045.4.3.2":   "ecdsa-with-SHA256",
	"1.2.840.10045.4.3.3":   "ecdsa-with-SHA384",
	"1.2.840.10045.4.3.4":   "ecdsa-with-SHA512",
	// EC named curves (relevant to CCC / digital key: P-256)
	"1.2.840.10045.3.1.7": "prime256v1 / secp256r1 / P-256",
	"1.3.132.0.34":        "secp384r1 / P-384",
	"1.3.132.0.35":        "secp521r1 / P-521",
	// Hash algorithms
	"2.16.840.1.101.3.4.2.1": "sha-256",
	"2.16.840.1.101.3.4.2.2": "sha-384",
	"2.16.840.1.101.3.4.2.3": "sha-512",
	// X.509 v3 extensions
	"2.5.29.14":         "subjectKeyIdentifier",
	"2.5.29.15":         "keyUsage",
	"2.5.29.17":         "subjectAltName",
	"2.5.29.19":         "basicConstraints",
	"2.5.29.31":         "cRLDistributionPoints",
	"2.5.29.35":         "authorityKeyIdentifier",
	"2.5.29.37":         "extKeyUsage",
	"1.3.6.1.5.5.7.1.1": "authorityInfoAccess",
	// Extended key usages
	"1.3.6.1.5.5.7.3.1": "serverAuth",
	"1.3.6.1.5.5.7.3.2": "clientAuth",
	// PKCS#7 / CMS content types
	"1.2.840.113549.1.7.1": "data",
	"1.2.840.113549.1.7.2": "signedData",
	"1.2.840.113549.1.7.3": "envelopedData",
}

func oidLookup(dotted string) string { return oidMap[dotted] }

var dottedRE = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)

func handleOID(raw json.RawMessage) (any, error) {
	var in pemInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	q := strings.TrimSpace(in.Input)
	if dottedRE.MatchString(q) {
		name := oidMap[q]
		if name == "" {
			name = "(not in local table)"
		}
		return map[string]any{"oid": q, "name": name}, nil
	}
	// search by name substring
	ql := strings.ToLower(q)
	var matches []map[string]string
	for oid, name := range oidMap {
		if strings.Contains(strings.ToLower(name), ql) {
			matches = append(matches, map[string]string{"oid": oid, "name": name})
		}
	}
	return map[string]any{"query": q, "matches": matches}, nil
}
