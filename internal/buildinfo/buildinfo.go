package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	Version   = "v0.1.0"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info holds full build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get returns the populated Info struct.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func (i Info) String() string {
	return fmt.Sprintf("Garund %s (%s) built at %s for %s", i.Version, i.Commit, i.BuildDate, i.Platform)
}
