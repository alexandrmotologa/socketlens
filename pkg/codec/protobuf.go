package codec

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"unicode/utf8"
)

// WireType represents a Protobuf wire encoding type.
type WireType int

const (
	WireVarint          WireType = 0
	WireFixed64         WireType = 1
	WireLengthDelimited WireType = 2
	WireStartGroup      WireType = 3 // Deprecated
	WireEndGroup        WireType = 4 // Deprecated
	WireFixed32         WireType = 5
)

// DecodeProtobuf parses raw Protobuf wire bytes into an arbitrary structured JSON object.
func DecodeProtobuf(data []byte) (string, error) {
	fields, err := parseProtoFields(data, 0)
	if err != nil {
		return "", err
	}

	prettyJSON, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting proto json: %w", err)
	}

	return string(prettyJSON), nil
}

// IsProbableProtobuf checks if the byte slice strictly complies with Protobuf wire format.
func IsProbableProtobuf(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// Must start with valid varint tag
	tag, n := binary.Uvarint(data)
	if n <= 0 {
		return false
	}

	fieldNum := tag >> 3
	wireType := WireType(tag & 0x07)

	if fieldNum == 0 || fieldNum > 536870911 {
		return false
	}

	switch wireType {
	case WireVarint, WireFixed64, WireLengthDelimited, WireFixed32:
		_, err := parseProtoFields(data, 0)
		return err == nil
	default:
		return false
	}
}

func parseProtoFields(data []byte, depth int) (map[string]interface{}, error) {
	if depth > 10 { // Recursion limit
		return map[string]interface{}{"raw_hex": hex.EncodeToString(data)}, nil
	}

	result := make(map[string]interface{})
	idx := 0

	for idx < len(data) {
		tag, n := binary.Uvarint(data[idx:])
		if n <= 0 {
			return nil, fmt.Errorf("invalid varint tag at index %d", idx)
		}
		idx += n

		fieldNum := tag >> 3
		wireType := WireType(tag & 0x07)

		if fieldNum == 0 {
			return nil, fmt.Errorf("invalid field number 0 at index %d", idx)
		}

		key := fmt.Sprintf("field_%d", fieldNum)

		switch wireType {
		case WireVarint:
			val, vn := binary.Uvarint(data[idx:])
			if vn <= 0 {
				return nil, fmt.Errorf("invalid varint value at index %d", idx)
			}
			idx += vn
			result[key] = val

		case WireFixed64:
			if idx+8 > len(data) {
				return nil, fmt.Errorf("unexpected EOF reading fixed64")
			}
			bits := binary.LittleEndian.Uint64(data[idx : idx+8])
			idx += 8
			// Heuristic: check if float or integer
			fval := math.Float64frombits(bits)
			if !math.IsNaN(fval) && !math.IsInf(fval, 0) && math.Abs(fval) > 1e-15 && math.Abs(fval) < 1e15 {
				result[key] = fval
			} else {
				result[key] = bits
			}

		case WireLengthDelimited:
			length, ln := binary.Uvarint(data[idx:])
			if ln <= 0 || idx+ln+int(length) > len(data) {
				return nil, fmt.Errorf("unexpected EOF reading length-delimited")
			}
			idx += ln
			payload := data[idx : idx+int(length)]
			idx += int(length)

			// 1. Try parsing as UTF-8 string
			if utf8.Valid(payload) && isPrintableASCII(payload) {
				result[key] = string(payload)
			} else {
				// 2. Try parsing as nested Protobuf message
				nested, err := parseProtoFields(payload, depth+1)
				if err == nil && len(nested) > 0 {
					result[key] = nested
				} else {
					result[key] = hex.EncodeToString(payload)
				}
			}

		case WireFixed32:
			if idx+4 > len(data) {
				return nil, fmt.Errorf("unexpected EOF reading fixed32")
			}
			bits := binary.LittleEndian.Uint32(data[idx : idx+4])
			idx += 4
			fval := math.Float32frombits(bits)
			if !math.IsNaN(float64(fval)) && !math.IsInf(float64(fval), 0) && math.Abs(float64(fval)) > 1e-6 && math.Abs(float64(fval)) < 1e6 {
				result[key] = fval
			} else {
				result[key] = bits
			}

		default:
			return nil, fmt.Errorf("unsupported wire type: %d", wireType)
		}
	}

	return result, nil
}
