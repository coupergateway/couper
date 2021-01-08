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
	APIErrTpl    *errors.Template
	FileErrTpl   *errors.Template
	ServerErrTpl *errors.Template
	APIBasePath  string
	FileBasePath string
	SPABasePath  string
	ServerName   string
}

func NewServerOptions(conf *config.Server) (*Options, error) {
	options := &Options{
		APIErrTpl:    errors.DefaultJSON,
		FileErrTpl:   errors.DefaultHTML,
		ServerErrTpl: errors.DefaultHTML,
	}

	if conf == nil {
		return options, nil
	}
	options.ServerName = conf.Name

	if conf.ErrorFile != "" {
		tpl, err := errors.NewTemplateFromFile(conf.ErrorFile)
		if err != nil {
			return nil, err
		}
		options.ServerErrTpl = tpl
		options.FileErrTpl = tpl
	}

	if conf.API != nil {
		options.APIBasePath = path.Join("/", conf.BasePath, conf.API.BasePath)

		if conf.API.ErrorFile != "" {
			tpl, err := errors.NewTemplateFromFile(conf.API.ErrorFile)
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

	if conf.Spa != nil {
		options.SPABasePath = utils.JoinPath("/", conf.BasePath, conf.Spa.BasePath)
	}

	return options, nil
}
