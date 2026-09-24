package codec

import (
	"encoding/json"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// DecodeCBOR decodes a CBOR payload into a formatted JSON string.
func DecodeCBOR(data []byte) (string, error) {
	var val interface{}
	err := cbor.Unmarshal(data, &val)
	if err != nil {
		return "", fmt.Errorf("decoding cbor: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting json: %w", err)
	}

	return string(prettyJSON), nil
}

// EncodeCBOR encodes a value into CBOR bytes.
func EncodeCBOR(val interface{}) ([]byte, error) {
	return cbor.Marshal(val)
}
