package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ReadPID reads the PID from the PID file.
func ReadPID(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pid format in %s: %w", pidFile, err)
	}
	return pid, nil
}

// WritePID writes the current or specified PID to the PID file.
func WritePID(pidFile string, pid int) error {
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// RemovePID removes the PID file.
func RemovePID(pidFile string) error {
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsProcessAlive checks if a process with the given PID is alive.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, Signal 0 tests process existence without killing it.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// StopProcess sends SIGTERM to the process and waits for it to exit.
func StopProcess(pid int) error {
	if !IsProcessAlive(pid) {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Send SIGTERM
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to process %d: %w", pid, err)
	}

	// Wait up to 5 seconds for termination
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsProcessAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Fallback to SIGKILL if process didn't terminate cleanly
	_ = proc.Signal(syscall.SIGKILL)
	return nil
}
