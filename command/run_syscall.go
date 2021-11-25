// +build linux darwin

package command

import (
	"syscall"

	"github.com/sirupsen/logrus"
)

func init() {
	checkLimit = func(logEntry *logrus.Entry) {
		lim := syscall.Rlimit{}
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
		if err != nil {
			logEntry.Warnf("ulimit: error retrieving file descriptor limit")
		} else {
			logEntry.Infof("ulimit: max open files: %d (hard limit: %d)", lim.Cur, lim.Max)
		}
	}
}
