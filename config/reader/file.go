package reader

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/avenga/couper/errors"
)

func ReadFromAttrFileJSONObjectOptional(context string, attributeValue map[string][]string, path string) (map[string][]string, error) {
	readErr := errors.Configuration.Label(context + ": read error")
	if attributeValue != nil && path != "" {
		return nil, readErr.Message("configured attribute and file")
	}

	if path != "" {
		fileContent, err := ReadFromFile(context, path)
		if err != nil {
			return nil, err
		}
		var m map[string][]string
		if err = json.Unmarshal(fileContent, &m); err != nil {
			return nil, readErr.Message("invalid file content")
		}
		return m, nil
	}

	return attributeValue, nil
}

func ReadFromAttrFile(context, attribute, path string) ([]byte, error) {
	readErr := errors.Configuration.Label(context + ": read error")
	if attribute != "" && path != "" {
		return nil, readErr.Message("configured attribute and file")
	} else if attribute == "" && path == "" {
		return nil, readErr.Message("required: configured attribute or file")
	}

	if path != "" {
		return ReadFromFile(context, path)
	}

	return []byte(attribute), nil
}

func ReadFromFile(context, path string) ([]byte, error) {
	readErr := errors.Configuration.Label(context + ": read error")
	if path == "" {
		return nil, readErr.Message("required: configured file")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, readErr.With(err)
	}
	b, err := os.ReadFile(absPath)
	if err != nil {
		return nil, readErr.With(err)
	}
	if len(b) == 0 {
		return nil, readErr.Message("empty file")
	}
	return b, nil
}
