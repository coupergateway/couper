package runtime

import "time"

var (
	BuildDate   = time.Now().Format("2006-01-02")
	BuildName   = "dev"
	VersionName = "0"
)
