package runtime

import (
	"net/http"
	"path"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/utils"
)

type MuxOptions struct {
	APIErrTpl      *errors.Template
	APIPath        string
	EndpointRoutes map[string]http.Handler
	FileBasePath   string
	FileErrTpl     *errors.Template
	FileRoutes     map[string]http.Handler
	SPARoutes      map[string]http.Handler
}

func NewMuxOptions(conf *config.Server) (*MuxOptions, error) {
	options := &MuxOptions{
		APIErrTpl:      errors.DefaultJSON,
		FileErrTpl:     errors.DefaultHTML,
		EndpointRoutes: make(map[string]http.Handler),
		FileRoutes:     make(map[string]http.Handler),
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

		options.FileBasePath = utils.JoinPath("/", conf.BasePath, conf.Files.BasePath)
	}

	return options, nil
}
