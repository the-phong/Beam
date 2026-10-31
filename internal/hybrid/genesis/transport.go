// Package b1genesis transports the distinct B1 genesis binary through
// CometBFT's JSON app_state without accepting a B0 transport fallback.
package genesis

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"

	b1state "example.com/pluto/internal/hybrid/state"
)

const MaxBinarySize = b1state.MaxGenesisSize
const MaxBase64Size = (MaxBinarySize + 2) / 3 * 4
const MaxJSONSize = MaxBase64Size + 2 + 64

var ErrTransport = errors.New("invalid B1 Genesis V3 JSON/Base64 transport")

func Encode(binary []byte, chainID string) ([]byte, error) {
	if _, err := b1state.ParseGenesis(binary, chainID); err != nil {
		return nil, ErrTransport
	}
	encoded := base64.StdEncoding.EncodeToString(binary)
	return json.Marshal(encoded)
}

func Decode(raw []byte, chainID string) ([]byte, b1state.State, error) {
	if len(raw) < 2 || len(raw) > MaxJSONSize {
		return nil, b1state.State{}, ErrTransport
	}
	trimmed := bytes.Trim(raw, " \t\n\r")
	if len(raw)-len(trimmed) > 64 || len(trimmed) < 2 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return nil, b1state.State{}, ErrTransport
	}
	var encoded string
	if err := json.Unmarshal(trimmed, &encoded); err != nil || len(encoded) == 0 || len(encoded) > MaxBase64Size {
		return nil, b1state.State{}, ErrTransport
	}
	binary, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(binary) > MaxBinarySize || base64.StdEncoding.EncodeToString(binary) != encoded {
		return nil, b1state.State{}, ErrTransport
	}
	canonical, err := json.Marshal(encoded)
	if err != nil || !bytes.Equal(trimmed, canonical) {
		return nil, b1state.State{}, ErrTransport
	}
	state, err := b1state.ParseGenesis(binary, chainID)
	if err != nil {
		return nil, b1state.State{}, ErrTransport
	}
	return binary, state, nil
}
