package assets

import (
	"bytes"
	"errors"
	"io"
)

var _ io.WriterTo = &AssetFile{}

type AssetFile struct {
	bytes []byte
	ct    string
	size  string
}

type Box struct {
	files map[string]*AssetFile
}

// Assets is referenced by the servers file serving.
var Assets *Box

func (af *AssetFile) Bytes() []byte {
	return af.bytes[:]
}

func (af *AssetFile) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(af.bytes)
	return int64(n), err
}

func (af *AssetFile) Size() string {
	return af.size
}

func (af *AssetFile) CT() string {
	return af.ct
}

func New() *Box {
	return &Box{
		files: map[string]*AssetFile{},
	}
}

func (b *Box) Open(file string) (*AssetFile, error) {
	if f, ok := b.files[file]; ok {
		return f, nil
	}
	return nil, errors.New("file not found: " + file)
}

func (b *Box) MustOpen(file string) *AssetFile {
	f, err := b.Open(file)
	if err != nil {
		return &AssetFile{bytes: []byte("not found")}
	}
	return f
}

func (af *AssetFile) String() string {
	buf := &bytes.Buffer{}
	_, _ = af.WriteTo(buf)
	return buf.String()
}
