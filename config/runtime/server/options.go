package server

import (
	"path"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/utils"
)

type Context interface {
	Options() *Options
}

type Options struct {
	APIErrTpl    []*errors.Template
	FileErrTpl   *errors.Template
	ServerErrTpl *errors.Template
	APIBasePath  []string
	FileBasePath string
	SPABasePath  string
	SrvBasePath  string
	ServerName   string
}

func NewServerOptions(conf *config.Server) (*Options, error) {
	options := &Options{
		APIErrTpl:    []*errors.Template{errors.DefaultJSON},
		FileErrTpl:   errors.DefaultHTML,
		ServerErrTpl: errors.DefaultHTML,
	}

	if conf == nil {
		return options, nil
	}
	options.ServerName = conf.Name
	options.SrvBasePath = path.Join("/", conf.BasePath)

	if conf.ErrorFile != "" {
		tpl, err := errors.NewTemplateFromFile(conf.ErrorFile)
		if err != nil {
			return nil, err
		}
		options.ServerErrTpl = tpl
		options.FileErrTpl = tpl
	}

	if len(conf.APIs) > 0 {
		options.APIBasePath = make([]string, len(conf.APIs))
		options.APIErrTpl = make([]*errors.Template, len(conf.APIs))
	}

	for i, api := range conf.APIs {
		options.APIBasePath[i] = path.Join(options.SrvBasePath, api.BasePath)

		if api.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(api.ErrorFile)
			if err != nil {
				return nil, err
			}
			options.APIErrTpl[i] = tpl
		} else {
			options.APIErrTpl[i] = errors.DefaultJSON
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

		options.FileBasePath = utils.JoinPath(options.SrvBasePath, conf.Files.BasePath)
	}

	if conf.Spa != nil {
		options.SPABasePath = utils.JoinPath(options.SrvBasePath, conf.Spa.BasePath)
	}

	return options, nil
}
