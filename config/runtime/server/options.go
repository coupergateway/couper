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
	APIErrTpls    map[*config.API]*errors.Template
	FilesErrTpl   *errors.Template
	ServerErrTpl  *errors.Template
	APIBasePaths  map[*config.API]string
	FilesBasePath string
	SPABasePath   string
	SrvBasePath   string
	ServerName    string
}

func NewServerOptions(conf *config.Server, logger *logrus.Entry) (*Options, error) {
	options := &Options{
		FilesErrTpl:  errors.DefaultHTML,
		ServerErrTpl: errors.DefaultHTML,
	}

	if conf == nil {
		return options, nil
	}
	options.ServerName = conf.Name
	options.SrvBasePath = path.Join("/", conf.BasePath)

	if conf.ErrorFile != "" {
		tpl, err := errors.NewTemplateFromFile(conf.ErrorFile, logger)
		if err != nil {
			return nil, err
		}
		options.ServerErrTpl = tpl
		options.FilesErrTpl = tpl
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

	if conf.Files != nil {
		if conf.Files.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(conf.Files.ErrorFile, logger)
			if err != nil {
				return nil, err
			}
			options.FilesErrTpl = tpl
		}

		options.FilesBasePath = utils.JoinPath(options.SrvBasePath, conf.Files.BasePath)
	}

	if conf.Spa != nil {
		options.SPABasePath = utils.JoinPath(options.SrvBasePath, conf.Spa.BasePath)
	}

	return options, nil
}
