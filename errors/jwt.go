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
	ErrorMissingKey               = errors.New("either non-empty key_file or key must be specified")
	ErrorDecodingKey              = errors.New("cannot decode the key data")
	ErrorUnsupportedSigningMethod = errors.New("unsupported signing method")
)

type JwtSigningError struct {
	error
}

func NewJWTError(err error) *JwtSigningError {
	return &JwtSigningError{
		error: err,
	}
}

func (e *JwtSigningError) Error() string {
	return e.error.Error()
}

func ValidateJWTKey(algo, key, file string) ([]byte, error) {
	var keyData []byte

	algo = strings.TrimSpace(algo)
	key = strings.TrimSpace(key)
	file = strings.TrimSpace(file)

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
	}

	if len(keyData) == 0 {
		return keyData, NewJWTError(ErrorMissingKey)
	} else if strings.HasPrefix(algo, "RS") {
		if b, _ := pem.Decode(keyData); b == nil {
			return keyData, NewJWTError(ErrorDecodingKey)
		}
	}

	return keyData, nil
}
