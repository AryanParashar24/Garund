package cli

import (
	"fmt"
)

func RunStop() error {
	cfg := DefaultConfig()
	pidFile := GetPIDFilePath(cfg.RuntimeDir)

	pid, err := ReadPID(pidFile)
	if err != nil {
		fmt.Println("Garund is not running (no PID file found).")
		return nil
	}

	if !IsProcessAlive(pid) {
		_ = RemovePID(pidFile)
		fmt.Println("Garund is not running (stale PID file removed).")
		return nil
	}

	fmt.Printf("Stopping Garund (PID: %d)...\n", pid)
	if err := StopProcess(pid); err != nil {
		return fmt.Errorf("failed to stop process %d: %w", pid, err)
	}

	_ = RemovePID(pidFile)
	fmt.Println("✓ Garund stopped.")
	return nil
}
