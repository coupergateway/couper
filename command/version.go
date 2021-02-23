package command

import (
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
)

var _ Cmd = &Version{}

type Version struct{}

func NewVersion() *Version {
	return &Version{}
}

func (v Version) Execute(_ Args, _ *config.Couper, _ *logrus.Entry) error {
	println(runtime.VersionName + " " + runtime.BuildDate + " " + runtime.BuildName)
	return nil
}

func (v Version) Usage() string {
	return "couper version"
}
