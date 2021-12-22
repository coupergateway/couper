package command

import (
	"fmt"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ Cmd = &Verify{}

type Verify struct{}

func NewVerify() *Verify {
	return &Verify{}
}

func (v Verify) Execute(args Args, _ *config.Couper, logger *logrus.Entry) error {
	if len(args) != 1 {
		v.Usage()

		err := fmt.Errorf("invalid number of arguments given")
		logger.WithError(err).Error()

		return err
	}

	cf, err := configload.LoadFile(args[0])
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
	tmpMemStore := cache.New(logger, tmpStoreCh)
	defer close(tmpStoreCh)
	_, err = runtime.NewServerConfiguration(cf, logger, tmpMemStore)
	if err != nil {
		logger.WithError(err).Error()
	}

	return err
}

func (v Verify) Usage() {
	println("Usage of verify:\n  verify [-f <file>]	Verify the syntax of the given configuration file.")
}
