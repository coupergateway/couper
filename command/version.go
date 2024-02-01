package command

import (
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/utils"
)

var _ Cmd = &Version{}

type Version struct{}

func NewVersion() *Version {
	return &Version{}
}

func (v Version) Execute(_ Args, _ *config.Couper, _ *logrus.Entry) error {
	println(utils.VersionName + " " + utils.BuildDate + " " + utils.BuildName)
	println("go version " + runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH)
	return nil
}

func (v Version) Usage() {
	println("Usage of version:\n  version	Print current version and build information.")
}
