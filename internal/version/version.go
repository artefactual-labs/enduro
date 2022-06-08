package version

import "runtime"

var (
	Version   = "(dev-version)"
	GitCommit = "(dev-commit)"
	BuildTime = "(dev-buildtime)"
	GoVersion = runtime.Version()
)
