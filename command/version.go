package command

import (
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/utils"
)

var _ Cmd = &Version{}

type Version struct{}

func NewVersion() *Version {
	return &Version{}
}

func (v Version) Execute(_ Args, _ *config.Couper, _ *logrus.Entry) error {
	println(utils.VersionName + " " + utils.BuildDate + " " + utils.BuildName)
	return nil
}

func (v Version) Usage() {
	println("Usage of version:\n  version	Print current version and build information.")
}
