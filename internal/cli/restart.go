package cli

import (
	"fmt"
	"time"
)

func RunRestart(opts StartOptions) error {
	fmt.Println("Restarting Garund...")
	if err := RunStop(); err != nil {
		fmt.Printf("Warning during stop: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)
	return RunStart(opts)
}
