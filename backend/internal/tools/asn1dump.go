package tools

import (
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

func handleASN1(raw json.RawMessage) (any, error) {
	var in pemInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	der, format, err := DecodeToDER(in.Input, "")
	if err != nil {
		return nil, err
	}
	nodes, err := walkASN1(der, 0)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"detectedFormat": format,
		"tree":           nodes,
	}, nil
}

func walkASN1(data []byte, depth int) ([]map[string]any, error) {
	if depth > 40 {
		return nil, fmt.Errorf("nesting too deep")
	}
	var nodes []map[string]any
	rest := data
	for len(rest) > 0 {
		var rv asn1.RawValue
		var err error
		rest, err = asn1.Unmarshal(rest, &rv)
		if err != nil {
			return nil, fmt.Errorf("ASN.1 parse error: %w", err)
		}
		node := map[string]any{
			"class":       classNames[rv.Class],
			"tag":         rv.Tag,
			"tagName":     tagName(rv.Class, rv.Tag),
			"constructed": rv.IsCompound,
			"length":      len(rv.Bytes),
		}
		if rv.IsCompound {
			children, err := walkASN1(rv.Bytes, depth+1)
			if err != nil {
				return nil, err
			}
			node["children"] = children
		} else {
			node["value"] = interpretPrimitive(rv)
			node["hex"] = hex.EncodeToString(rv.Bytes)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

var classNames = map[int]string{0: "universal", 1: "application", 2: "context", 3: "private"}

func tagName(class, tag int) string {
	if class != 0 {
		return fmt.Sprintf("[%d]", tag)
	}
	names := map[int]string{
		1: "BOOLEAN", 2: "INTEGER", 3: "BIT STRING", 4: "OCTET STRING",
		5: "NULL", 6: "OBJECT IDENTIFIER", 10: "ENUMERATED", 12: "UTF8String",
		16: "SEQUENCE", 17: "SET", 19: "PrintableString", 20: "T61String",
		22: "IA5String", 23: "UTCTime", 24: "GeneralizedTime", 26: "VisibleString",
	}
	if n, ok := names[tag]; ok {
		return n
	}
	return fmt.Sprintf("tag-%d", tag)
}

func interpretPrimitive(rv asn1.RawValue) any {
	if rv.Class != 0 {
		return nil
	}
	switch rv.Tag {
	case 1: // BOOLEAN
		return len(rv.Bytes) > 0 && rv.Bytes[0] != 0
	case 2: // INTEGER
		i := new(big.Int).SetBytes(rv.Bytes)
		if len(rv.Bytes) > 0 && rv.Bytes[0]&0x80 != 0 { // negative
			i.Sub(i, new(big.Int).Lsh(big.NewInt(1), uint(len(rv.Bytes)*8)))
		}
		return i.String()
	case 6: // OID
		var oid asn1.ObjectIdentifier
		if _, err := asn1.Unmarshal(rv.FullBytes, &oid); err == nil {
			s := oid.String()
			if name := oidLookup(s); name != "" {
				return s + " (" + name + ")"
			}
			return s
		}
	case 12, 19, 22, 20, 26: // string types
		if isPrintable(rv.Bytes) {
			return string(rv.Bytes)
		}
	case 23, 24: // UTCTime / GeneralizedTime
		if isPrintable(rv.Bytes) {
			s := string(rv.Bytes)
			if t, err := parseASN1Time(rv.Tag, s); err == nil {
				return s + " (" + t.UTC().Format(time.RFC3339) + ")"
			}
			return s
		}
	case 5: // NULL
		return nil
	}
	return nil
}

func parseASN1Time(tag int, s string) (time.Time, error) {
	layouts := []string{"20060102150405Z0700", "20060102150405Z", "200601021504Z"}
	if tag == 23 {
		layouts = []string{"060102150405Z0700", "060102150405Z", "0601021504Z"}
	}
	var err error
	for _, l := range layouts {
		var t time.Time
		if t, err = time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return strings.TrimSpace(string(b)) != ""
}
