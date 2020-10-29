package runtime

import (
	"net/http"
	"path"
	"path/filepath"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/utils"
)

type MuxOptions struct {
	APIErrTpl      *errors.Template
	APIPath        string
	EndpointRoutes map[string]http.Handler
	FileBasePath   string
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
		options.APIPath = path.Join("/", conf.BasePath, conf.API.BasePath)

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
		options.FileBasePath = utils.JoinPath("/", conf.BasePath, conf.Files.BasePath)
		options.FileHandler = handler.NewFile(options.FileBasePath, absPath, options.FileErrTpl)
	}

	return options, nil
}
