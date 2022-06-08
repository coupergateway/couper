package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type File = struct {
	Path     string
	IsDir    bool
	Children Files
}

type Files []File

func NewFile(filePath string) File {
	return File{
		Path:  filePath,
		IsDir: false,
	}
}

func NewDir(filePath string) (File, error) {
	children, err := readDir(filePath)
	return File{
		Path:     filePath,
		IsDir:    true,
		Children: children,
	}, err
}

func NewFiles(filesList []string) ([]File, error) {
	var files []File

	for _, f := range filesList {
		filePath, err := filepath.Abs(f)
		if err != nil {
			return nil, err
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			dir, err := NewDir(filePath)
			if err != nil {
				return nil, err
			}

			files = append(files, dir)
		} else {
			files = append(files, NewFile(filePath))
		}
	}

	return files, nil
}

func (f *Files) Refresh() (*Files, error) {
	var result Files
	for _, file := range *f {
		if file.IsDir {
			dir, err := NewDir(file.Path)
			if err != nil {
				return nil, err
			}
			result = append(result, dir)
		} else {
			result = append(result, file)
		}
	}
	return &result, nil
}

func (f *Files) AsList() []string {
	var list []string
	for _, file := range *f {
		if file.IsDir {
			list = append(list, file.Children.AsList()...)
		} else {
			list = append(list, file.Path)
		}
	}
	return list
}

func readDir(filePath string) (Files, error) {
	// ReadDir ... returns a list ... sorted by filename.
	listing, err := ioutil.ReadDir(filePath)
	if err != nil {
		return nil, err
	}

	var entries Files
	for _, item := range listing {
		if item.IsDir() || filepath.Ext(item.Name()) != ".hcl" {
			continue
		}

		filename := filepath.Join(filePath, item.Name())
		entries = append(entries, NewFile(filename))
	}

	return entries, nil
}
