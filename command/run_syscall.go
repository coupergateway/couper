//go:build linux || darwin

package command

import (
	"syscall"

	"github.com/sirupsen/logrus"
)

func init() {
	limitFn = logRlimit
}

func logRlimit(logEntry *logrus.Entry) {
	lim := syscall.Rlimit{}
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
	if err != nil {
		logEntry.Warnf("ulimit: error retrieving file descriptor limit")
	} else {
		logEntry.Debugf("ulimit: max open files: %d (hard limit: %d)", lim.Cur, lim.Max)
	}
}
