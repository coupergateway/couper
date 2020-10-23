package runtime

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/utils"
)

type MuxOptions struct {
	APIErrTpl      *errors.Template
	APIPath        string
	EndpointRoutes map[string]http.Handler
	FileHandler    http.Handler
	FileErrTpl     *errors.Template
	SPARoutes      map[string]http.Handler
}

func NewMuxOptions(conf *config.Server) (*MuxOptions, error) {
	options := &MuxOptions{
		APIErrTpl:      errors.DefaultJSON,
		FileErrTpl:     errors.DefaultHTML,
		EndpointRoutes: make(map[string]http.Handler),
		SPARoutes:      make(map[string]http.Handler),
	}

	if conf.API != nil {
		options.APIPath = utils.JoinPath("/", conf.BasePath, conf.API.BasePath)
		for strings.HasSuffix(options.APIPath, "/") {
			options.APIPath = options.APIPath[:len(options.APIPath)-1]
		}

		if conf.API.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(conf.Files.ErrorFile)
			if err != nil {
				return nil, err
			}
			options.APIErrTpl = tpl
		}
	}

	if conf.Files != nil {
		if conf.Files.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(conf.Files.ErrorFile)
			if err != nil {
				return nil, err
			}
			options.FileErrTpl = tpl
		}

		absPath, err := filepath.Abs(conf.Files.DocumentRoot)
		if err != nil {
			return nil, err
		}
		options.FileHandler = handler.NewFile(conf.Files.BasePath, absPath, options.FileErrTpl)
	}

	return options, nil
}
