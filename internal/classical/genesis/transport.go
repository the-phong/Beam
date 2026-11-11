// Package b0genesis handles the D-025 CometBFT JSON/Base64 genesis transport.
// The decoded D-021 binary is the only source of B0 genesis state.
package b0genesis

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"

	"example.com/pluto/internal/classical/state"
)

const (
	MaxBinarySize = 8 + 4 + 2 + 50 + 4 + 28*b0state.MaxAccounts
	MaxBase64Size = (MaxBinarySize + 2) / 3 * 4
	// The canonical JSON string has two quotes; at most 64 surrounding
	// whitespace bytes are permitted, keeping parsing bounded.
	MaxJSONSize = MaxBase64Size + 2 + 64
)

var ErrTransport = errors.New("invalid B0 Genesis V2 JSON/Base64 transport")

// Encode validates canonical Genesis V2 binary before wrapping it in one
// JSON string containing RFC 4648 standard padded Base64.
func Encode(binary []byte, expectedChainID string) ([]byte, error) {
	if _, err := b0state.ParseGenesis(binary, expectedChainID); err != nil {
		return nil, ErrTransport
	}
	encoded := base64.StdEncoding.EncodeToString(binary)
	value, err := json.Marshal(encoded)
	if err != nil || len(value) > MaxJSONSize {
		return nil, ErrTransport
	}
	return value, nil
}

// Decode rejects alternate JSON and Base64 spellings before invoking the
// sole Genesis V2 binary parser. Whitespace around the JSON value is allowed.
func Decode(raw []byte, expectedChainID string) ([]byte, b0state.State, error) {
	if len(raw) < 2 || len(raw) > MaxJSONSize {
		return nil, b0state.State{}, ErrTransport
	}
	trimmed := trimJSONWhitespace(raw)
	if len(raw)-len(trimmed) > 64 || len(trimmed) < 2 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return nil, b0state.State{}, ErrTransport
	}
	var encoded string
	if err := json.Unmarshal(trimmed, &encoded); err != nil || len(encoded) == 0 || len(encoded) > MaxBase64Size {
		return nil, b0state.State{}, ErrTransport
	}
	// Strict rejects nonzero trailing pad bits; re-encoding rejects alternate
	// alphabets, padding and all other noncanonical Base64 representations.
	binary, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(binary) > MaxBinarySize || base64.StdEncoding.EncodeToString(binary) != encoded {
		return nil, b0state.State{}, ErrTransport
	}
	canonicalJSON, err := json.Marshal(encoded)
	if err != nil || !bytes.Equal(trimmed, canonicalJSON) {
		return nil, b0state.State{}, ErrTransport
	}
	state, err := b0state.ParseGenesis(binary, expectedChainID)
	if err != nil {
		return nil, b0state.State{}, ErrTransport
	}
	return binary, state, nil
}

func trimJSONWhitespace(raw []byte) []byte {
	start, end := 0, len(raw)
	for start < end && jsonWhitespace(raw[start]) {
		start++
	}
	for end > start && jsonWhitespace(raw[end-1]) {
		end--
	}
	return raw[start:end]
}

func jsonWhitespace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }
