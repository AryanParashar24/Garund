package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/garund/garund/internal/buildinfo"
	k8s "github.com/garund/garund/internal/kubernetes"
	"github.com/garund/garund/internal/web"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type DoctorCheck struct {
	Name   string
	Passed bool
	Detail string
	Remedy string
}

func RunDoctor() {
	cfg := DefaultConfig()
	info := buildinfo.Get()

	fmt.Printf("\nGarund Doctor\n")
	fmt.Printf("─────────────────────────────\n\n")

	var checks []DoctorCheck

	// 1. Binary & Version
	checks = append(checks, DoctorCheck{
		Name:   "Binary & Version",
		Passed: true,
		Detail: fmt.Sprintf("Garund %s (%s)", info.Version, info.Platform),
	})

	// 2. Runtime directory
	if err := EnsureRuntimeDirs(cfg.RuntimeDir); err == nil {
		checks = append(checks, DoctorCheck{
			Name:   "Runtime Directory",
			Passed: true,
			Detail: cfg.RuntimeDir,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Runtime Directory",
			Passed: false,
			Detail: fmt.Sprintf("Cannot create/write to %s: %v", cfg.RuntimeDir, err),
			Remedy: "Ensure home directory write permissions.",
		})
	}

	// 3. Log directory
	logFile := GetLogFilePath(cfg.RuntimeDir)
	if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		_ = f.Close()
		checks = append(checks, DoctorCheck{
			Name:   "Log Directory",
			Passed: true,
			Detail: logFile,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Log Directory",
			Passed: false,
			Detail: fmt.Sprintf("Log file unwritable: %v", err),
			Remedy: fmt.Sprintf("Check permissions on %s", logFile),
		})
	}

	// 4. Kubeconfig
	if _, err := os.Stat(cfg.Kubeconfig); err == nil {
		checks = append(checks, DoctorCheck{
			Name:   "kubeconfig",
			Passed: true,
			Detail: cfg.Kubeconfig,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "kubeconfig",
			Passed: false,
			Detail: fmt.Sprintf("Kubeconfig not found at %s", cfg.Kubeconfig),
			Remedy: "Export KUBECONFIG or place config at ~/.kube/config",
		})
	}

	// 5. Kubernetes Context & API
	k8sClient, err := k8s.NewClient(cfg.Kubeconfig)
	if err == nil && k8sClient != nil {
		ctxName := "default"
		if rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.Kubeconfig},
			&clientcmd.ConfigOverrides{},
		).RawConfig(); err == nil {
			ctxName = rawConfig.CurrentContext
		}

		// Validate API access by listing namespaces with 3s timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, apiErr := k8sClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		cancel()

		if apiErr == nil {
			checks = append(checks, DoctorCheck{
				Name:   "Kubernetes Context & API",
				Passed: true,
				Detail: fmt.Sprintf("Context: %s (API reachable)", ctxName),
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   "Kubernetes Context & API",
				Passed: false,
				Detail: fmt.Sprintf("Context: %s (API unreachable: %v)", ctxName, apiErr),
				Remedy: "Ensure cluster is running and kubectl context is valid.",
			})
		}
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Kubernetes Context & API",
			Passed: false,
			Detail: fmt.Sprintf("Failed to initialize Kubernetes client: %v", err),
			Remedy: "Verify kubeconfig content.",
		})
	}

	// 6. Port Check
	addr := GetAddr(cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		_ = listener.Close()
		checks = append(checks, DoctorCheck{
			Name:   fmt.Sprintf("Port %d", cfg.Port),
			Passed: true,
			Detail: fmt.Sprintf("Port %d is available on %s", cfg.Port, cfg.Host),
		})
	} else {
		// Check if active Garund is listening
		pidFile := GetPIDFilePath(cfg.RuntimeDir)
		pid, _ := ReadPID(pidFile)
		if IsProcessAlive(pid) {
			checks = append(checks, DoctorCheck{
				Name:   fmt.Sprintf("Port %d", cfg.Port),
				Passed: true,
				Detail: fmt.Sprintf("In use by active Garund process (PID %d)", pid),
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   fmt.Sprintf("Port %d", cfg.Port),
				Passed: false,
				Detail: fmt.Sprintf("Port %d is occupied by another process", cfg.Port),
				Remedy: fmt.Sprintf("Stop the process occupying port %d or run garund start --port %d", cfg.Port, cfg.Port+1),
			})
		}
	}

	// 7. Backend & Dashboard Health
	url := fmt.Sprintf("http://%s/health", addr)
	client := http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(url); err == nil && resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		checks = append(checks, DoctorCheck{
			Name:   "Backend & Dashboard Health",
			Passed: true,
			Detail: "HTTP /health responding 200 OK",
		})
	} else {
		pidFile := GetPIDFilePath(cfg.RuntimeDir)
		pid, _ := ReadPID(pidFile)
		if IsProcessAlive(pid) {
			checks = append(checks, DoctorCheck{
				Name:   "Backend & Dashboard Health",
				Passed: false,
				Detail: "Process running but HTTP /health did not respond 200 OK",
				Remedy: "Check garund logs using 'garund logs'",
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   "Backend & Dashboard Health",
				Passed: true,
				Detail: "Server not running (run 'garund start' to launch)",
			})
		}
	}

	// 8. Frontend Assets
	if web.HasAssets() {
		checks = append(checks, DoctorCheck{
			Name:   "Frontend Assets",
			Passed: true,
			Detail: "Embedded production dashboard assets present",
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Frontend Assets",
			Passed: false,
			Detail: "Embedded static frontend assets missing or corrupted",
			Remedy: "Rebuild binary using 'make build' or reinstall Garund",
		})
	}

	// Output summary
	hasFailures := false
	for _, c := range checks {
		if c.Passed {
			fmt.Printf("✓ %-28s %s\n", c.Name, c.Detail)
		} else {
			hasFailures = true
			fmt.Printf("✗ %-28s %s\n", c.Name, c.Detail)
			if c.Remedy != "" {
				fmt.Printf("  └─ Remedy: %s\n", c.Remedy)
			}
		}
	}

	fmt.Println()
	if hasFailures {
		fmt.Println("Some issues were detected. Please review the remedies above.")
	} else {
		fmt.Println("No problems detected.")
	}
}
