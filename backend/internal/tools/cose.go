package tools

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func handleCOSE(raw json.RawMessage) (any, error) {
	var in struct {
		Input       string `json:"input"`
		InputFormat string `json:"inputFormat"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	b, detected, err := decodeFlexibleBytes(in.Input, orDefault(in.InputFormat, "auto"))
	if err != nil {
		return nil, err
	}
	d := &cborDecoder{data: b}
	val, err := d.decode(0)
	if err != nil {
		return nil, err
	}

	out := map[string]any{"detectedInput": detected, "cbor": val}
	coseType := ""
	if m, ok := val.(map[string]any); ok {
		if tag, ok := m["_tag"]; ok {
			coseType = coseTagName(fmt.Sprintf("%v", tag))
			val = m["value"]
		}
	}

	switch v := val.(type) {
	case []any:
		if len(v) == 4 {
			out["interpretation"] = interpretSign1(v)
			if coseType == "" {
				coseType = "COSE_Sign1 (by shape)"
			}
		}
	case map[string]any:
		out["interpretation"] = interpretCOSEKey(v)
		if coseType == "" {
			coseType = "COSE_Key (by shape)"
		}
	}
	if coseType != "" {
		out["coseType"] = coseType
	}
	return out, nil
}

func coseTagName(tag string) string {
	switch tag {
	case "18":
		return "COSE_Sign1"
	case "98":
		return "COSE_Sign"
	case "17":
		return "COSE_Mac0"
	case "16":
		return "COSE_Encrypt0"
	}
	return "tag " + tag
}

var coseAlg = map[string]string{
	"-7": "ES256", "-35": "ES384", "-36": "ES512",
	"-8": "EdDSA", "-37": "PS256", "-38": "PS384", "-39": "PS512",
	"5": "HMAC 256/256",
}
var coseKty = map[string]string{"1": "OKP", "2": "EC2", "3": "RSA", "4": "Symmetric"}
var coseCrv = map[string]string{"1": "P-256", "2": "P-384", "3": "P-521", "6": "Ed25519"}

func interpretSign1(v []any) map[string]any {
	res := map[string]any{
		"protectedHeaderRaw": v[0],
		"unprotectedHeader":  v[1],
		"payload":            v[2],
		"signature":          v[3],
	}
	// protected header is a bstr -> "h'<hex>'" wrapping a CBOR map
	if s, ok := v[0].(string); ok {
		if raw := unwrapBstr(s); raw != nil {
			dd := &cborDecoder{data: raw}
			if ph, err := dd.decode(0); err == nil {
				res["protectedHeader"] = annotateHeader(ph)
			}
		}
	}
	return res
}

func annotateHeader(h any) any {
	m, ok := h.(map[string]any)
	if !ok {
		return h
	}
	out := map[string]any{}
	for k, val := range m {
		switch k {
		case "1": // alg
			out["alg"] = nameOr(coseAlg, val)
		case "4":
			out["kid"] = val
		default:
			out[k] = val
		}
	}
	return out
}

func interpretCOSEKey(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, val := range m {
		switch k {
		case "1":
			out["kty"] = nameOr(coseKty, val)
		case "2":
			out["kid"] = val
		case "3":
			out["alg"] = nameOr(coseAlg, val)
		case "-1":
			out["crv"] = nameOr(coseCrv, val)
		case "-2":
			out["x"] = val
		case "-3":
			out["y"] = val
		default:
			out["label("+k+")"] = val
		}
	}
	return out
}

func nameOr(m map[string]string, v any) string {
	key := fmt.Sprintf("%v", v)
	if name, ok := m[key]; ok {
		return fmt.Sprintf("%v (%s)", v, name)
	}
	return key
}

func unwrapBstr(s string) []byte {
	if strings.HasPrefix(s, "h'") && strings.HasSuffix(s, "'") {
		b, err := hex.DecodeString(s[2 : len(s)-1])
		if err == nil {
			return b
		}
	}
	return nil
}
