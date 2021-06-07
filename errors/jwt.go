package errors

import (
	"encoding/pem"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
)

var (
	ErrorNoProfileForLabel        = errors.New("no signing profile for label")
	ErrorMissingKey               = errors.New("either key_file or key must be specified")
	ErrorDecodingKey              = errors.New("cannot decode the key data")
	ErrorUnsupportedSigningMethod = errors.New("unsupported signing method")
	ErrorEmptyKeyFile             = errors.New("empty key_file")
	ErrorEmptyKey                 = errors.New("empty key")
)

func LoadJWTKey(algo, key, file string) ([]byte, error) {
	var keyData []byte

	if file != "" {
		p, err := filepath.Abs(file)
		if err != nil {
			return keyData, err
		}
		content, err := ioutil.ReadFile(p)
		if err != nil {
			return keyData, err
		}

		keyData = content
	} else if key != "" {
		keyData = []byte(key)
	} else {
		return keyData, ErrorMissingKey
	}

	if len(keyData) == 0 {
		if file != "" {
			return keyData, ErrorEmptyKeyFile
		}

		return keyData, ErrorEmptyKey
	} else if strings.HasPrefix(algo, "RS") {
		if b, _ := pem.Decode(keyData); b == nil {
			return keyData, ErrorDecodingKey
		}
	}

	return keyData, nil
}
