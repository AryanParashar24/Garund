package cli

import (
	"fmt"

	"github.com/garund/garund/internal/buildinfo"
)

func RunVersion() {
	info := buildinfo.Get()
	fmt.Printf("Garund %s\n", info.Version)
	fmt.Printf("  commit:   %s\n", info.Commit)
	fmt.Printf("  build:    %s\n", info.BuildDate)
	fmt.Printf("  platform: %s\n", info.Platform)
	fmt.Printf("  go:       %s\n", info.GoVersion)
}
