package tools

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type tlvInput struct {
	Input string `json:"input"`
	Mode  string `json:"mode"` // tlv | apdu-command | apdu-response
}

func handleTLV(raw json.RawMessage) (any, error) {
	var in tlvInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	b, err := hex.DecodeString(wsRE.ReplaceAllString(strings.TrimSpace(in.Input), ""))
	if err != nil {
		return nil, fmt.Errorf("input must be hex: %w", err)
	}
	switch in.Mode {
	case "apdu-command":
		return parseAPDUCommand(b)
	case "apdu-response":
		return parseAPDUResponse(b)
	default:
		nodes, err := parseTLV(b, 0)
		if err != nil {
			return nil, err
		}
		return map[string]any{"tlv": nodes}, nil
	}
}

func parseTLV(data []byte, depth int) ([]map[string]any, error) {
	if depth > 30 {
		return nil, errors.New("TLV nesting too deep")
	}
	var nodes []map[string]any
	i := 0
	for i < len(data) {
		tagStart := i
		if data[i]&0x1f == 0x1f { // multi-byte tag
			i++
			for i < len(data) && data[i]&0x80 != 0 {
				i++
			}
			i++
		} else {
			i++
		}
		if i > len(data) {
			return nil, errors.New("truncated tag")
		}
		tagBytes := data[tagStart:i]
		constructed := tagBytes[0]&0x20 != 0

		if i >= len(data) {
			return nil, errors.New("missing length")
		}
		l0 := data[i]
		i++
		var length int
		if l0&0x80 == 0 {
			length = int(l0)
		} else {
			n := int(l0 & 0x7f)
			if n == 0 {
				return nil, errors.New("indefinite length (0x80) is not supported")
			}
			if n > 4 {
				return nil, fmt.Errorf("length field of %d bytes is too long", n)
			}
			if i+n > len(data) {
				return nil, errors.New("truncated length")
			}
			var l uint64
			for k := 0; k < n; k++ {
				l = l<<8 | uint64(data[i])
				i++
			}
			if l > uint64(len(data)-i) {
				return nil, fmt.Errorf("value length %d exceeds remaining bytes", l)
			}
			length = int(l)
		}
		if i+length > len(data) {
			return nil, fmt.Errorf("value length %d exceeds remaining bytes", length)
		}
		value := data[i : i+length]
		i += length

		node := map[string]any{
			"tag":         strings.ToUpper(hex.EncodeToString(tagBytes)),
			"constructed": constructed,
			"length":      length,
		}
		if constructed {
			children, err := parseTLV(value, depth+1)
			if err != nil {
				return nil, err
			}
			node["children"] = children
		} else {
			node["value"] = strings.ToUpper(hex.EncodeToString(value))
			if isPrintable(value) {
				node["ascii"] = string(value)
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

var insNames = map[byte]string{
	0xA4: "SELECT", 0xB0: "READ BINARY", 0xC0: "GET RESPONSE",
	0x20: "VERIFY", 0x88: "INTERNAL AUTHENTICATE", 0x82: "EXTERNAL AUTHENTICATE",
	0x84: "GET CHALLENGE", 0xCA: "GET DATA", 0xDA: "PUT DATA",
}

func parseAPDUCommand(b []byte) (any, error) {
	if len(b) < 4 {
		return nil, errors.New("command APDU needs at least 4 bytes (CLA INS P1 P2)")
	}
	out := map[string]any{
		"cla":     hexByte(b[0]),
		"ins":     hexByte(b[1]),
		"insName": insNames[b[1]],
		"p1":      hexByte(b[2]),
		"p2":      hexByte(b[3]),
	}
	switch {
	case len(b) == 4:
		out["case"] = "1 (no data, no response expected)"
	case len(b) == 5:
		out["case"] = "2 (no data, Le present)"
		out["le"] = apduLe(b[4])
	default:
		lc := int(b[4])
		if 5+lc > len(b) {
			return nil, errors.New("Lc exceeds APDU length")
		}
		out["lc"] = lc
		out["data"] = strings.ToUpper(hex.EncodeToString(b[5 : 5+lc]))
		if rem := b[5+lc:]; len(rem) == 1 {
			out["case"] = "4 (data + Le)"
			out["le"] = apduLe(rem[0])
		} else {
			out["case"] = "3 (data, no Le)"
		}
	}
	return out, nil
}

var swNames = map[string]string{
	"9000": "Success",
	"6700": "Wrong length",
	"6982": "Security status not satisfied",
	"6985": "Conditions of use not satisfied",
	"6A82": "File or application not found",
	"6A86": "Incorrect P1/P2",
	"6D00": "Instruction not supported",
	"6E00": "Class not supported",
}

func parseAPDUResponse(b []byte) (any, error) {
	if len(b) < 2 {
		return nil, errors.New("response APDU needs at least 2 bytes (SW1 SW2)")
	}
	sw := strings.ToUpper(hex.EncodeToString(b[len(b)-2:]))
	out := map[string]any{
		"data":       strings.ToUpper(hex.EncodeToString(b[:len(b)-2])),
		"sw":         sw,
		"sw1":        hexByte(b[len(b)-2]),
		"sw2":        hexByte(b[len(b)-1]),
		"statusWord": swNames[sw],
	}
	if out["statusWord"] == "" {
		out["statusWord"] = "(unknown / see ISO 7816-4)"
	}
	return out, nil
}

func hexByte(b byte) string { return fmt.Sprintf("%02X", b) }

// apduLe interprets a short-form Le byte: 0x00 means "up to 256 bytes"
// (ISO 7816-4), not zero.
func apduLe(b byte) int {
	if b == 0 {
		return 256
	}
	return int(b)
}
