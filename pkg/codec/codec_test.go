package codec

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

func TestMsgPackCodec(t *testing.T) {
	inputMap := map[string]interface{}{
		"action": "subscribe",
		"id":     int64(42),
		"pairs":  []interface{}{"BTC/USD", "ETH/USD"},
	}

	encoded, err := EncodeMsgPack(inputMap)
	if err != nil {
		t.Fatalf("msgpack encode error: %v", err)
	}

	format := SniffFormat(encoded)
	if format != client.FormatMsgPack {
		t.Errorf("expected format msgpack, got %s", format)
	}

	decoded, err := DecodeMsgPack(encoded)
	if err != nil {
		t.Fatalf("msgpack decode error: %v", err)
	}

	if !strings.Contains(decoded, "subscribe") || !strings.Contains(decoded, "BTC/USD") {
		t.Errorf("unexpected decoded string: %s", decoded)
	}
}

func TestCBORCodec(t *testing.T) {
	inputMap := map[string]interface{}{
		"status": "active",
		"code":   int64(200),
	}

	encoded, err := EncodeCBOR(inputMap)
	if err != nil {
		t.Fatalf("cbor encode error: %v", err)
	}

	format := SniffFormat(encoded)
	if format != client.FormatCBOR {
		t.Errorf("expected format cbor, got %s", format)
	}

	decoded, err := DecodeCBOR(encoded)
	if err != nil {
		t.Fatalf("cbor decode error: %v", err)
	}

	if !strings.Contains(decoded, "active") || !strings.Contains(decoded, "200") {
		t.Errorf("unexpected decoded string: %s", decoded)
	}
}

func TestProtobufWireDecoder(t *testing.T) {
	// Synthesize a valid Protobuf binary payload:
	// field 1 (varint): 150 -> tag = (1 << 3) | 0 = 8 -> 0x08, value = 150 -> 0x96, 0x01
	// field 2 (length-delimited): "testing" -> tag = (2 << 3) | 2 = 18 -> 0x12, len = 7, "testing"
	var buf []byte

	// Field 1: varint 150
	buf = binary.AppendUvarint(buf, (1<<3)|0)
	buf = binary.AppendUvarint(buf, 150)

	// Field 2: string "testing"
	str := "testing"
	buf = binary.AppendUvarint(buf, (2<<3)|2)
	buf = binary.AppendUvarint(buf, uint64(len(str)))
	buf = append(buf, []byte(str)...)

	if !IsProbableProtobuf(buf) {
		t.Fatal("expected IsProbableProtobuf to return true")
	}

	decoded, err := DecodeProtobuf(buf)
	if err != nil {
		t.Fatalf("protobuf decode error: %v", err)
	}

	if !strings.Contains(decoded, "field_1") || !strings.Contains(decoded, "150") {
		t.Errorf("expected field_1 = 150, got %s", decoded)
	}
	if !strings.Contains(decoded, "field_2") || !strings.Contains(decoded, "testing") {
		t.Errorf("expected field_2 = testing, got %s", decoded)
	}
}

func TestGzipPipeline(t *testing.T) {
	rawJSON := []byte(`{"event":"quote","symbol":"NVDA","price":125.5}`)

	compressed, err := CompressGzip(rawJSON)
	if err != nil {
		t.Fatalf("gzip compress error: %v", err)
	}

	format := SniffFormat(compressed)
	if format != client.FormatGzip {
		t.Errorf("expected format gzip, got %s", format)
	}

	fmtName, decoded, err := DecodeFrame(compressed)
	if err != nil {
		t.Fatalf("decode frame error: %v", err)
	}

	if fmtName != "gzip+json" {
		t.Errorf("expected format gzip+json, got %s", fmtName)
	}
	if !strings.Contains(decoded, "NVDA") {
		t.Errorf("expected NVDA in decoded body, got %s", decoded)
	}
}

func TestHexDump(t *testing.T) {
	raw := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}
	dump := HexDump(raw)
	if !strings.Contains(dump, "00000000") || !strings.Contains(dump, "ff fe fd") {
		t.Errorf("unexpected hex dump: %s", dump)
	}
}
