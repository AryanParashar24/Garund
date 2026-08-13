package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/garund/garund/internal/server"
)

func RunLogs() error {
	cfg := DefaultConfig()
	logFile := GetLogFilePath(cfg.RuntimeDir)

	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No log file found at %s\n", logFile)
			return nil
		}
		return fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	defer f.Close()

	fmt.Printf("Garund Logs (%s)\n", logFile)
	fmt.Printf("─────────────────────────────────────────\n")

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			sanitized := server.SanitizeErrorMessage(line)
			fmt.Print(sanitized)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}
