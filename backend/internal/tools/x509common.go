package tools

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strings"
)

func pkixNameToMap(n pkix.Name) map[string]any {
	m := map[string]any{}
	put := func(k string, v []string) {
		if len(v) == 1 {
			m[k] = v[0]
		} else if len(v) > 1 {
			m[k] = v
		}
	}
	put("commonName", str1(n.CommonName))
	put("organization", n.Organization)
	put("organizationalUnit", n.OrganizationalUnit)
	put("country", n.Country)
	put("province", n.Province)
	put("locality", n.Locality)
	put("streetAddress", n.StreetAddress)
	put("postalCode", n.PostalCode)
	put("serialNumber", str1(n.SerialNumber))
	if s := n.String(); s != "" {
		m["dn"] = s
	}
	return m
}

func str1(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func publicKeyInfo(pub any) map[string]any {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return map[string]any{
			"algorithm": "RSA",
			"keySize":   k.N.BitLen(),
			"exponent":  k.E,
			"modulus":   "0x" + strings.ToUpper(k.N.Text(16)),
		}
	case *ecdsa.PublicKey:
		byteLen := (k.Curve.Params().BitSize + 7) / 8
		return map[string]any{
			"algorithm": "ECDSA",
			"curve":     k.Curve.Params().Name,
			"keySize":   k.Curve.Params().BitSize,
			"x":         "0x" + strings.ToUpper(k.X.Text(16)),
			"y":         "0x" + strings.ToUpper(k.Y.Text(16)),
			"point":     "04" + hex.EncodeToString(padLeft(k.X.Bytes(), byteLen)) + hex.EncodeToString(padLeft(k.Y.Bytes(), byteLen)),
		}
	case ed25519.PublicKey:
		return map[string]any{
			"algorithm": "Ed25519",
			"keySize":   256,
			"publicKey": hex.EncodeToString(k),
		}
	default:
		return map[string]any{"algorithm": fmt.Sprintf("%T", pub)}
	}
}

func padLeft(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}

func sansToMap(dns []string, emails []string, ips []net.IP, uris []*url.URL) map[string]any {
	m := map[string]any{}
	if len(dns) > 0 {
		m["dnsNames"] = dns
	}
	if len(emails) > 0 {
		m["emailAddresses"] = emails
	}
	if len(ips) > 0 {
		s := make([]string, len(ips))
		for i, ip := range ips {
			s[i] = ip.String()
		}
		m["ipAddresses"] = s
	}
	if len(uris) > 0 {
		s := make([]string, len(uris))
		for i, u := range uris {
			s[i] = u.String()
		}
		m["uris"] = s
	}
	return m
}
