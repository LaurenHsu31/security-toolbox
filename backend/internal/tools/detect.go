package tools

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	hexOnly = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	wsRE    = regexp.MustCompile(`\s+`)
	// Matches a BEGIN marker even if the surrounding dashes are the wrong count
	// or were normalized from Unicode dashes. The trailing space is what real
	// base64 (which has no spaces) won't accidentally contain.
	beginRE = regexp.MustCompile(`(?i)-+\s*BEGIN[ \t]`)
	// Matches a BEGIN/END marker (any dash count) so the base64 body can be
	// recovered even when the whole PEM is on a single line.
	pemMarkerRE = regexp.MustCompile(`(?i)-+\s*(?:BEGIN|END)[^-]*-+`)
)

// literalEscapes turns literal "\n" / "\r" / "\t" two-character escape
// sequences (the way a PEM looks after being embedded in JSON or a log line)
// into real whitespace. base64/hex/PEM never legitimately contain a backslash,
// so this is safe to apply to any input.
var literalEscapes = strings.NewReplacer(
	`\r\n`, "\n",
	`\n`, "\n",
	`\r`, "\n",
	`\t`, " ",
)

// normalizeDashes rewrites Unicode dash/hyphen variants (en/em dash,
// non-breaking hyphen U+2011, minus sign, …) to ASCII '-'. Copy-pasting a PEM
// from a PDF or word processor frequently mangles the "-----" markers this way.
func normalizeDashes(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' {
			return r
		}
		if unicode.Is(unicode.Dash, r) || unicode.Is(unicode.Hyphen, r) {
			return '-'
		}
		return r
	}, s)
}

// lenientPEMBody recovers the base64 payload from PEM-ish text whose markers are
// malformed (wrong dash count, stray characters) or whose whole content sits on
// a single line, so pem.Decode rejected it. It strips the BEGIN/END markers and
// any dashes/whitespace, then base64-decodes what remains.
func lenientPEMBody(s string) ([]byte, bool) {
	stripped := pemMarkerRE.ReplaceAllString(s, "")
	var b strings.Builder
	for _, r := range stripped {
		// PEM bodies are standard base64 (never base64url), so a stray '-' can
		// only be a leftover marker dash — drop it.
		if r != '-' && !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	compact := b.String()
	if compact == "" {
		return nil, false
	}
	for _, dec := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if der, err := dec.DecodeString(compact); err == nil && len(der) > 0 {
			return der, true
		}
	}
	return nil, false
}

// DecodeToDER turns user input into raw DER bytes and reports the detected
// input format. Detection order: PEM -> Hex -> Base64/Base64URL.
//
// expectPEMType, when non-empty (e.g. "CERTIFICATE REQUEST"), is only used to
// pick the right block when several PEM blocks are present.
func DecodeToDER(input, expectPEMType string) (der []byte, format string, err error) {
	trimmed := strings.TrimSpace(normalizeDashes(literalEscapes.Replace(input)))
	if trimmed == "" {
		return nil, "", errors.New("input is empty")
	}

	if beginRE.MatchString(trimmed) {
		// Strict PEM first: handles well-formed input and the expected type
		// when several blocks are present.
		rest := []byte(trimmed)
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if expectPEMType == "" || block.Type == expectPEMType ||
				strings.Contains(block.Type, expectPEMType) {
				return block.Bytes, "PEM (" + block.Type + ")", nil
			}
		}
		// Markers exist but pem.Decode rejected them (e.g. wrong dash count from
		// a mangled copy-paste). Recover the base64 body directly.
		if b, ok := lenientPEMBody(trimmed); ok {
			return b, "PEM (recovered from malformed markers)", nil
		}
		return nil, "", errors.New("PEM markers found but the block could not be parsed")
	}

	compact := wsRE.ReplaceAllString(trimmed, "")

	// Pure hex (even length) is treated as hex; this almost never collides with
	// real base64 of binary, which contains non-hex characters or padding.
	if len(compact)%2 == 0 && hexOnly.MatchString(compact) {
		if b, e := hex.DecodeString(compact); e == nil {
			return b, "Hex", nil
		}
	}

	if b, e := base64.StdEncoding.DecodeString(compact); e == nil {
		return b, "Base64", nil
	}
	if b, e := base64.RawStdEncoding.DecodeString(compact); e == nil {
		return b, "Base64 (unpadded)", nil
	}
	if b, e := base64.URLEncoding.DecodeString(compact); e == nil {
		return b, "Base64URL", nil
	}
	if b, e := base64.RawURLEncoding.DecodeString(compact); e == nil {
		return b, "Base64URL (unpadded)", nil
	}

	return nil, "", errors.New("could not detect input format (expected PEM, DER in Base64, or Hex)")
}

// decodeFlexibleBytes decodes text/hex/base64 input into raw bytes for the
// generic encoders and hashers.
func decodeFlexibleBytes(input, inputFormat string) ([]byte, string, error) {
	switch strings.ToLower(inputFormat) {
	case "", "auto":
		// fallthrough to detection below
	case "utf8", "text":
		return []byte(input), "UTF-8", nil
	case "hex":
		b, e := hex.DecodeString(wsRE.ReplaceAllString(input, ""))
		return b, "Hex", e
	case "base64":
		b, e := base64.StdEncoding.DecodeString(wsRE.ReplaceAllString(input, ""))
		return b, "Base64", e
	case "base64url":
		b, e := base64.RawURLEncoding.DecodeString(strings.TrimRight(wsRE.ReplaceAllString(input, ""), "="))
		return b, "Base64URL", e
	}
	// auto
	compact := wsRE.ReplaceAllString(strings.TrimSpace(input), "")
	if len(compact)%2 == 0 && len(compact) > 0 && hexOnly.MatchString(compact) {
		if b, e := hex.DecodeString(compact); e == nil {
			return b, "Hex", nil
		}
	}
	if b, e := base64.StdEncoding.DecodeString(compact); e == nil {
		return b, "Base64", nil
	}
	if b, e := base64.RawURLEncoding.DecodeString(strings.TrimRight(compact, "=")); e == nil {
		return b, "Base64URL", nil
	}
	return []byte(input), "UTF-8", nil
}
