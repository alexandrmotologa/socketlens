package codec

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// DecompressGzip inflates a Gzip-compressed byte slice.
func DecompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer reader.Close()

	out, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading gzip payload: %w", err)
	}

	return out, nil
}

// CompressGzip deflates raw bytes using standard Gzip compression.
func CompressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("writing gzip payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
