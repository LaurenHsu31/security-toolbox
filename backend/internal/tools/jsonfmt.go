package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type jsonInput struct {
	Input  string `json:"input"`
	Mode   string `json:"mode"`   // beautify | minify | validate
	Indent flexInt `json:"indent"` // spaces, default 2
}

func handleJSON(raw json.RawMessage) (any, error) {
	var in jsonInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	src := []byte(in.Input)

	if !json.Valid(src) {
		// Re-parse to extract a precise position.
		var v any
		err := json.Unmarshal(src, &v)
		if se, ok := err.(*json.SyntaxError); ok {
			line, col := lineCol(in.Input, int(se.Offset))
			return nil, fmt.Errorf("invalid JSON at line %d, column %d: %s", line, col, se.Error())
		}
		if err != nil {
			return nil, fmt.Errorf("invalid JSON: %s", err.Error())
		}
		return nil, fmt.Errorf("invalid JSON")
	}

	mode := in.Mode
	if mode == "" {
		mode = "beautify"
	}
	indent := int(in.Indent)
	if indent <= 0 {
		indent = 2
	}

	var buf bytes.Buffer
	switch mode {
	case "minify":
		if err := json.Compact(&buf, src); err != nil {
			return nil, err
		}
	case "validate":
		return map[string]any{"valid": true, "mode": "validate"}, nil
	default: // beautify
		if err := json.Indent(&buf, src, "", strings.Repeat(" ", indent)); err != nil {
			return nil, err
		}
		// Also hand back the parsed value so the UI can render a collapsible
		// tree. UseNumber keeps integers/decimals exact instead of float64.
		dec := json.NewDecoder(bytes.NewReader(src))
		dec.UseNumber()
		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return nil, err
		}
		return map[string]any{
			"valid":     true,
			"mode":      mode,
			"formatted": buf.String(),
			"parsed":    parsed,
		}, nil
	}
	return map[string]any{
		"valid":     true,
		"mode":      mode,
		"formatted": buf.String(),
	}, nil
}

func lineCol(s string, offset int) (int, int) {
	if offset > len(s) {
		offset = len(s)
	}
	line, col := 1, 1
	// Count runes, not bytes, so columns stay right after multi-byte characters.
	for i, r := range s {
		if i >= offset {
			break
		}
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
