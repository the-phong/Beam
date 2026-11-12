// Package researchv2genesis carries the binary research genesis through
// CometBFT's canonical JSON app_state field.
package researchv2genesis

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"

	"example.com/pluto/internal/experiment/profiles"
)

const MaxBinarySize = 8 + 4 + 2 + researchv2.MaxChainID + 4 + researchv2.MaxGenesisAccounts*(20+8+33+mldsa44.PublicKeySize)
const MaxBase64Size = (MaxBinarySize + 2) / 3 * 4
const MaxJSONSize = MaxBase64Size + 66

var ErrTransport = errors.New("invalid research-v2 genesis transport")

func Encode(binary []byte, chainID string) ([]byte, error) {
	if _, err := researchv2.ParseGenesis(binary, chainID); err != nil {
		return nil, ErrTransport
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(binary))
}

func Decode(raw []byte, chainID string) ([]byte, researchv2.State, error) {
	if len(raw) < 2 || len(raw) > MaxJSONSize {
		return nil, researchv2.State{}, ErrTransport
	}
	trimmed := bytes.Trim(raw, " \t\n\r")
	if len(raw)-len(trimmed) > 64 || len(trimmed) < 2 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return nil, researchv2.State{}, ErrTransport
	}
	var encoded string
	if err := json.Unmarshal(trimmed, &encoded); err != nil || len(encoded) == 0 || len(encoded) > MaxBase64Size {
		return nil, researchv2.State{}, ErrTransport
	}
	binary, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(binary) > MaxBinarySize || base64.StdEncoding.EncodeToString(binary) != encoded {
		return nil, researchv2.State{}, ErrTransport
	}
	canonical, err := json.Marshal(encoded)
	if err != nil || !bytes.Equal(trimmed, canonical) {
		return nil, researchv2.State{}, ErrTransport
	}
	state, err := researchv2.ParseGenesis(binary, chainID)
	if err != nil {
		return nil, researchv2.State{}, ErrTransport
	}
	return binary, state, nil
}
