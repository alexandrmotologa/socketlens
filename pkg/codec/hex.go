package codec

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// HexDump generates a standard 16-byte columnar hex dump with ASCII sidebar.
func HexDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return hex.Dump(data)
}

// FormatJSON takes raw JSON bytes and returns indented JSON with 2 spaces.
func FormatJSON(data []byte) (string, error) {
	var buf bytes.Buffer
	err := json.Indent(&buf, bytes.TrimSpace(data), "", "  ")
	if err != nil {
		return "", fmt.Errorf("indenting json: %w", err)
	}
	return buf.String(), nil
}

// DecodeFrame processes a raw byte buffer through the full codec pipeline.
func DecodeFrame(data []byte) (format string, decoded string, err error) {
	if len(data) == 0 {
		return "text", "", nil
	}

	detected := SniffFormat(data)

	// If Gzip compressed, decompress first
	if detected == "gzip" {
		decompressed, decErr := DecompressGzip(data)
		if decErr == nil {
			innerFmt, innerDec, innerErr := DecodeFrame(decompressed)
			if innerErr == nil {
				return fmt.Sprintf("gzip+%s", innerFmt), innerDec, nil
			}
			return "gzip", HexDump(decompressed), nil
		}
		return "gzip", HexDump(data), decErr
	}

	switch detected {
	case "json":
		formatted, fErr := FormatJSON(data)
		if fErr == nil {
			return "json", formatted, nil
		}
		return "json", string(data), nil

	case "msgpack":
		formatted, mErr := DecodeMsgPack(data)
		if mErr == nil {
			return "msgpack", formatted, nil
		}
		return "msgpack", HexDump(data), mErr

	case "cbor":
		formatted, cErr := DecodeCBOR(data)
		if cErr == nil {
			return "cbor", formatted, nil
		}
		return "cbor", HexDump(data), cErr

	case "protobuf":
		formatted, pErr := DecodeProtobuf(data)
		if pErr == nil {
			return "protobuf", formatted, nil
		}
		return "protobuf", HexDump(data), pErr

	case "text":
		return "text", strings.TrimRight(string(data), "\r\n"), nil

	default:
		return "raw", HexDump(data), nil
	}
}
