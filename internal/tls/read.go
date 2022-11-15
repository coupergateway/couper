package tls

import (
	"os"

	"github.com/avenga/couper/errors"
)

func ReadValueOrFile(value, filePath string) ([]byte, error) {
	if value != "" && filePath != "" {
		return nil, errors.Configuration.Message("both attributes provided: value and file")
	} else if value != "" {
		return []byte(value), nil
	} else if filePath != "" {
		return os.ReadFile(filePath)
	}
	return nil, errors.Configuration.Message("no attributes provided")
}
