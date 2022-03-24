package utils

import "time"

var (
	BuildDate   = ""
	BuildName   = "dev"
	VersionName = "0"
)

func init() {
	if BuildDate == "" {
		BuildDate = time.Now().Format("2006-01-02")
	}

	// strip out possible semver v
	if len(VersionName) > 0 && VersionName[0] == 'v' {
		VersionName = VersionName[1:]
	} else if VersionName == "master" {
		VersionName = "edge"
	}
}
