package command

import (
	"context"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/runtime"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ Cmd = &Verify{}

type Verify struct{}

func NewVerify() *Verify {
	return &Verify{}
}

func (v Verify) Execute(args Args, conf *config.Couper, logger *logrus.Entry) error {
	cf, err := configload.LoadFiles(args, conf.Environment)
	if diags, ok := err.(hcl.Diagnostics); ok {
		for _, diag := range diags {
			logger.WithError(diag).Error()
		}

		return diags
	} else if err != nil {
		logger.WithError(err).Error()

		return err
	}

	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)

	tmpMemStore := cache.New(logger, tmpStoreCh)

	ctx, cancel := context.WithCancel(cf.Context)
	cf.Context = ctx
	defer cancel()

	_, err = runtime.NewServerConfiguration(cf, logger, tmpMemStore)
	if err != nil {
		logger.WithError(err).Error()
	}

	return err
}

func (v Verify) Usage() {
	println("Usage of verify:\n  verify [-f <file>]	[-d <dir>]	Verify the syntax of the given configuration file(s).")
}
