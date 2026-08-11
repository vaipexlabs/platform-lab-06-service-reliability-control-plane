// Package document normalizes the YAML and JSON documents accepted by the
// reliability control-plane interfaces.
package document

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"gopkg.in/yaml.v3"
)

// Normalize converts exactly one YAML or JSON document into JSON bytes.
func Normalize(data []byte) ([]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("document is empty")
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple YAML or JSON documents are not allowed")
		}
		return nil, err
	}
	return json.Marshal(value)
}
