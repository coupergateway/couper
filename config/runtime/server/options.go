package server

import (
	"path"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/utils"
)

type Context interface {
	Options() *Options
}

type Options struct {
	APIErrTpls     map[*config.API]*errors.Template
	FilesErrTpls   []*errors.Template
	ServerErrTpl   *errors.Template
	APIBasePaths   map[*config.API]string
	FilesBasePaths []string
	SPABasePaths   []string
	SrvBasePath    string
	ServerName     string
	TLS            *config.ServerTLS
}

func NewServerOptions(conf *config.Server, logger *logrus.Entry) (*Options, error) {
	options := &Options{
		ServerErrTpl: errors.DefaultHTML,
	}

	if conf == nil {
		return options, nil
	}

	options.TLS = conf.TLS

	options.FilesErrTpls = make([]*errors.Template, len(conf.Files))
	for i := range conf.Files {
		options.FilesErrTpls[i] = errors.DefaultHTML
	}

	options.ServerName = conf.Name
	options.SrvBasePath = path.Join("/", conf.BasePath)

	if conf.ErrorFile != "" {
		tpl, err := errors.NewTemplateFromFile(conf.ErrorFile, logger)
		if err != nil {
			return nil, err
		}

		options.ServerErrTpl = tpl
		for i := range conf.Files {
			options.FilesErrTpls[i] = tpl
		}
	}

	if len(conf.APIs) > 0 {
		options.APIBasePaths = make(map[*config.API]string)
		options.APIErrTpls = make(map[*config.API]*errors.Template)
	}

	for _, api := range conf.APIs {
		options.APIBasePaths[api] = path.Join(options.SrvBasePath, api.BasePath)

		if api.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(api.ErrorFile, logger)
			if err != nil {
				return nil, err
			}
			options.APIErrTpls[api] = tpl
		} else {
			options.APIErrTpls[api] = errors.DefaultJSON
		}
	}

	options.FilesBasePaths = make([]string, len(conf.Files))
	for i, f := range conf.Files {
		if f.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(f.ErrorFile, logger)
			if err != nil {
				return nil, err
			}

			options.FilesErrTpls[i] = tpl
		}

		options.FilesBasePaths[i] = utils.JoinOpenAPIPath(options.SrvBasePath, f.BasePath)
	}

	for _, s := range conf.SPAs {
		options.SPABasePaths = append(options.SPABasePaths, utils.JoinOpenAPIPath(options.SrvBasePath, s.BasePath))
	}

	return options, nil
}
