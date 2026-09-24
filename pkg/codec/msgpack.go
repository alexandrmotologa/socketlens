package codec

import (
	"encoding/json"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// DecodeMsgPack decodes a MessagePack payload into formatted JSON string.
func DecodeMsgPack(data []byte) (string, error) {
	var val interface{}
	err := msgpack.Unmarshal(data, &val)
	if err != nil {
		return "", fmt.Errorf("decoding msgpack: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting json: %w", err)
	}

	return string(prettyJSON), nil
}

// EncodeMsgPack encodes a JSON-compatible value into MessagePack bytes.
func EncodeMsgPack(val interface{}) ([]byte, error) {
	return msgpack.Marshal(val)
}
