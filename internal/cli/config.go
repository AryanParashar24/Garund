package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Kubeconfig    string `json:"kubeconfig"`
	Context       string `json:"context"`
	PrometheusURL string `json:"prometheusUrl"`
	RuntimeDir    string `json:"runtimeDir"`
}

func DefaultConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	runtimeDir := os.Getenv("GARUND_HOME")
	if runtimeDir == "" {
		runtimeDir = filepath.Join(home, ".garund")
	}

	return Config{
		Host:          "127.0.0.1",
		Port:          8080,
		Kubeconfig:    kubeconfig,
		Context:       "",
		PrometheusURL: os.Getenv("PROMETHEUS_URL"),
		RuntimeDir:    runtimeDir,
	}
}

func EnsureRuntimeDirs(runtimeDir string) error {
	dirs := []string{
		runtimeDir,
		filepath.Join(runtimeDir, "config"),
		filepath.Join(runtimeDir, "logs"),
		filepath.Join(runtimeDir, "run"),
		filepath.Join(runtimeDir, "data"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create runtime directory %s: %w", d, err)
		}
	}
	return nil
}

func GetPIDFilePath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "run", "garund.pid")
}

func GetLogFilePath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "logs", "garund.log")
}

func GetAddr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
