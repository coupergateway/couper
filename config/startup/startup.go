package startup

import (
	"os"
	"path/filepath"
)

// SetWorkingDirectory sets the working directory.
func SetWorkingDirectory(configFile string) (string, error) {
	if err := os.Chdir(filepath.Dir(configFile)); err != nil {
		return "", err
	}

	return os.Getwd()
}
