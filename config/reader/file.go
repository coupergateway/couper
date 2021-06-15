package reader

import (
	"io/ioutil"
	"path/filepath"

	"github.com/avenga/couper/errors"
)

func ReadFromAttrFile(context, attribute, path string) ([]byte, error) {
	readErr := errors.Configuration.Label(context + ": read error")
	if attribute != "" && path != "" {
		return nil, readErr.Message("configured attribute and file")
	}

	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadFile(absPath)
		if err != nil {
			return nil, readErr.With(err)
		}
		if len(b) == 0 {
			return nil, readErr.Message("empty file")
		}
		return b, nil
	}

	if attribute != "" {
		return []byte(attribute), nil
	} else {
		return nil, readErr.Message("empty attribute")
	}
}
