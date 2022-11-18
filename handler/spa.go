package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/server/writer"
)

var (
	_ http.Handler   = &Spa{}
	_ server.Context = &Spa{}
)

type Spa struct {
	bootstrapContent []byte
	bootstrapModTime time.Time
	bootstrapOnce    sync.Once
	config           *config.Spa
	modifier         []hcl.Body
	srvOptions       *server.Options
}

func NewSpa(ctx *hcl.EvalContext, config *config.Spa, srvOpts *server.Options, modifier []hcl.Body) (*Spa, error) {
	var err error
	if config.BootstrapFile, err = filepath.Abs(config.BootstrapFile); err != nil {
		return nil, err
	}

	spa := &Spa{
		config:     config,
		modifier:   modifier,
		srvOptions: srvOpts,
	}

	if config.BootstrapData == nil {
		return spa, nil
	}

	file, err := os.Open(config.BootstrapFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	spa.bootstrapModTime = fileInfo.ModTime()

	spa.replaceBootstrapData(ctx, file)

	return spa, nil
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var content io.ReadSeeker
	var modTime time.Time

	if len(s.bootstrapContent) > 0 {
		content = strings.NewReader(string(s.bootstrapContent))
		modTime = s.bootstrapModTime
	} else {
		file, err := os.Open(s.config.BootstrapFile)
		if err != nil {
			if _, ok := err.(*os.PathError); ok {
				s.srvOptions.ServerErrTpl.WithError(errors.RouteNotFound).ServeHTTP(rw, req)
				return
			}

			s.srvOptions.ServerErrTpl.WithError(errors.Configuration).ServeHTTP(rw, req)
			return
		}
		content = file
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil || fileInfo.IsDir() {
			s.srvOptions.ServerErrTpl.WithError(errors.Configuration).ServeHTTP(rw, req)
			return
		}

		modTime = fileInfo.ModTime()
	}

	evalContext := eval.ContextFromRequest(req)

	if r, ok := rw.(*writer.Response); ok {
		r.AddModifier(evalContext.HCLContext(), s.modifier...)
	}

	http.ServeContent(rw, req, s.config.BootstrapFile, modTime, content)
}

func (s *Spa) replaceBootstrapData(ctx *hcl.EvalContext, reader io.ReadCloser) {
	if s.config.BootstrapData == nil {
		return
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	val, _ := s.config.BootstrapData.Value(ctx)
	data, err := ctyjson.Marshal(val, val.Type())
	if err != nil {
		panic(err)
	}

	escapedData := &bytes.Buffer{}
	json.HTMLEscape(escapedData, data)

	const defaultName = "__BOOTSTRAP_DATA__"
	bootstrapName := s.config.BootStrapDataName
	if bootstrapName == "" {
		bootstrapName = defaultName
	}
	s.bootstrapContent = bytes.Replace(b, []byte(bootstrapName), escapedData.Bytes(), 1)
}

func (s *Spa) Options() *server.Options {
	return s.srvOptions
}

func (s *Spa) String() string {
	return "spa"
}
