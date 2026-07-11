package tools

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
)

type cborDecoder struct {
	data []byte
	pos  int
}

func handleCBOR(raw json.RawMessage) (any, error) {
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
	return map[string]any{
		"detectedInput": detected,
		"decoded":       val,
		"trailingBytes": len(b) - d.pos,
	}, nil
}

func (d *cborDecoder) decode(depth int) (any, error) {
	if depth > 64 {
		return nil, errors.New("CBOR nesting too deep")
	}
	if d.pos >= len(d.data) {
		return nil, errors.New("unexpected end of CBOR data")
	}
	b := d.data[d.pos]
	mt := b >> 5
	ai := b & 0x1f
	d.pos++

	switch mt {
	case 0: // unsigned int
		return d.readUint(ai)
	case 1: // negative int
		v, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		u := toUint64(v)
		if u <= math.MaxInt64 {
			return -1 - int64(u), nil
		}
		return new(big.Int).Sub(big.NewInt(-1), new(big.Int).SetUint64(u)), nil
	case 2: // byte string
		bs, err := d.readBytes(ai, depth)
		if err != nil {
			return nil, err
		}
		return "h'" + hex.EncodeToString(bs) + "'", nil
	case 3: // text string
		bs, err := d.readBytes(ai, depth)
		if err != nil {
			return nil, err
		}
		return string(bs), nil
	case 4: // array
		return d.readArray(ai, depth)
	case 5: // map
		return d.readMap(ai, depth)
	case 6: // tag
		tag, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		inner, err := d.decode(depth + 1)
		if err != nil {
			return nil, err
		}
		return map[string]any{"_tag": toUint64(tag), "value": inner}, nil
	case 7: // simple / float
		return d.readSimple(ai)
	}
	return nil, fmt.Errorf("unsupported CBOR major type %d", mt)
}

func (d *cborDecoder) readUint(ai byte) (any, error) {
	switch {
	case ai < 24:
		return uint64(ai), nil
	case ai == 24:
		return d.readN(1)
	case ai == 25:
		return d.readN(2)
	case ai == 26:
		return d.readN(4)
	case ai == 27:
		return d.readN(8)
	}
	return nil, fmt.Errorf("invalid additional info %d for integer", ai)
}

func (d *cborDecoder) readN(n int) (uint64, error) {
	if d.pos+n > len(d.data) {
		return 0, errors.New("unexpected end while reading integer")
	}
	var v uint64
	for i := 0; i < n; i++ {
		v = v<<8 | uint64(d.data[d.pos])
		d.pos++
	}
	return v, nil
}

func toUint64(v any) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	}
	return 0
}

func (d *cborDecoder) readBytes(ai byte, depth int) ([]byte, error) {
	if ai == 31 { // indefinite-length
		if depth > 64 {
			return nil, errors.New("CBOR nesting too deep")
		}
		// Chunks are raw definite-length strings; read their headers directly so
		// byte-string chunks are concatenated as bytes (decode() would render
		// them as "h'…'" display strings).
		var out []byte
		for {
			if d.pos >= len(d.data) {
				return nil, errors.New("unterminated indefinite-length string")
			}
			hb := d.data[d.pos]
			if hb == 0xff {
				d.pos++
				break
			}
			mt, cai := hb>>5, hb&0x1f
			if (mt != 2 && mt != 3) || cai == 31 {
				return nil, errors.New("indefinite-length string chunks must be definite-length strings")
			}
			d.pos++
			chunk, err := d.readBytes(cai, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, chunk...)
		}
		return out, nil
	}
	n, err := d.readUint(ai)
	if err != nil {
		return nil, err
	}
	u := toUint64(n)
	if u > uint64(len(d.data)-d.pos) {
		return nil, errors.New("byte/text length exceeds data")
	}
	ln := int(u)
	out := d.data[d.pos : d.pos+ln]
	d.pos += ln
	return out, nil
}

func (d *cborDecoder) readArray(ai byte, depth int) (any, error) {
	var arr []any
	if ai == 31 {
		for {
			if d.pos < len(d.data) && d.data[d.pos] == 0xff {
				d.pos++
				break
			}
			v, err := d.decode(depth + 1)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	}
	n, err := d.readUint(ai)
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < toUint64(n); i++ {
		v, err := d.decode(depth + 1)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	return arr, nil
}

func (d *cborDecoder) readMap(ai byte, depth int) (any, error) {
	out := map[string]any{}
	add := func() error {
		k, err := d.decode(depth + 1)
		if err != nil {
			return err
		}
		v, err := d.decode(depth + 1)
		if err != nil {
			return err
		}
		out[fmt.Sprintf("%v", k)] = v
		return nil
	}
	if ai == 31 {
		for {
			if d.pos < len(d.data) && d.data[d.pos] == 0xff {
				d.pos++
				break
			}
			if err := add(); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	n, err := d.readUint(ai)
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < toUint64(n); i++ {
		if err := add(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (d *cborDecoder) readSimple(ai byte) (any, error) {
	switch ai {
	case 20:
		return false, nil
	case 21:
		return true, nil
	case 22, 23:
		return nil, nil
	case 24:
		return d.readN(1)
	case 25:
		v, err := d.readN(2)
		if err != nil {
			return nil, err
		}
		return float16to64(uint16(v)), nil
	case 26:
		v, err := d.readN(4)
		if err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(uint32(v))), nil
	case 27:
		v, err := d.readN(8)
		if err != nil {
			return nil, err
		}
		return math.Float64frombits(v), nil
	}
	return nil, fmt.Errorf("unsupported simple value %d", ai)
}

func float16to64(h uint16) float64 {
	sign := uint64(h>>15) & 0x1
	exp := uint64(h>>10) & 0x1f
	mant := uint64(h) & 0x3ff
	var bits uint64
	switch {
	case exp == 0:
		if mant == 0 {
			bits = sign << 63
		} else { // subnormal
			exp = 1023 - 15 + 1
			for mant&0x400 == 0 {
				mant <<= 1
				exp--
			}
			mant &= 0x3ff
			bits = sign<<63 | exp<<52 | mant<<42
		}
	case exp == 0x1f:
		bits = sign<<63 | 0x7ff<<52 | mant<<42
	default:
		bits = sign<<63 | (exp-15+1023)<<52 | mant<<42
	}
	return math.Float64frombits(bits)
}
