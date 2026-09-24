package codec

import (
	"bytes"
	"encoding/json"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/fxamacker/cbor/v2"
	"github.com/vmihailenco/msgpack/v5"
)

// SniffFormat analyzes raw byte buffers to determine likely serialization format.
func SniffFormat(data []byte) client.PayloadFormat {
	if len(data) == 0 {
		return client.FormatText
	}

	// 1. Check Gzip magic bytes: 0x1F, 0x8B
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return client.FormatGzip
	}

	// 2. Check JSON
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '{' && last == '}') || (first == '[' && last == ']') {
			if json.Valid(trimmed) {
				return client.FormatJSON
			}
		}
	}

	// 3. Check MessagePack (maps or arrays typical for streaming messages)
	firstByte := data[0]
	isMsgPackLeading := (firstByte >= 0x80 && firstByte <= 0x8f) || // fixmap
		(firstByte >= 0x90 && firstByte <= 0x9f) || // fixarray
		firstByte == 0xdc || firstByte == 0xdd || // array16, array32
		firstByte == 0xde || firstByte == 0xdf // map16, map32

	if isMsgPackLeading {
		var dummy interface{}
		if err := msgpack.Unmarshal(data, &dummy); err == nil && dummy != nil {
			return client.FormatMsgPack
		}
	}

	// 4. Check CBOR (maps or arrays: major type 4: 0x80-0x9b, major type 5: 0xa0-0xbb)
	isCBORLeading := (firstByte >= 0x80 && firstByte <= 0x9b) || (firstByte >= 0xa0 && firstByte <= 0xbb)
	if isCBORLeading {
		var dummy interface{}
		if err := cbor.Unmarshal(data, &dummy); err == nil && dummy != nil {
			return client.FormatCBOR
		}
	}

	// 5. Check Protobuf wire format
	if IsProbableProtobuf(data) {
		return client.FormatProtobuf
	}

	// 6. Check UTF-8 plain text
	if isPrintableASCII(data) {
		return client.FormatText
	}

	return client.FormatRaw
}

func isPrintableASCII(data []byte) bool {
	for _, b := range data {
		// Allow tab, newline, carriage return, and printable range 0x20..0x7E
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}
