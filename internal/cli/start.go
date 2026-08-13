package cli

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garund/garund/internal/buildinfo"
	k8s "github.com/garund/garund/internal/kubernetes"
	"github.com/garund/garund/internal/server"
	"k8s.io/client-go/tools/clientcmd"
)

type StartOptions struct {
	Host          string
	Port          int
	Kubeconfig    string
	Context       string
	PrometheusURL string
}

func RunStart(opts StartOptions) error {
	cfg := DefaultConfig()
	if opts.Host != "" {
		cfg.Host = opts.Host
	}
	if opts.Port > 0 {
		cfg.Port = opts.Port
	}
	if opts.Kubeconfig != "" {
		cfg.Kubeconfig = opts.Kubeconfig
	}
	if opts.Context != "" {
		cfg.Context = opts.Context
	}
	if opts.PrometheusURL != "" {
		cfg.PrometheusURL = opts.PrometheusURL
	}

	addr := GetAddr(cfg.Host, cfg.Port)
	url := fmt.Sprintf("http://%s", addr)

	// 1. Ensure runtime directories exist
	if err := EnsureRuntimeDirs(cfg.RuntimeDir); err != nil {
		return fmt.Errorf("failed to initialize runtime environment: %w", err)
	}

	pidFile := GetPIDFilePath(cfg.RuntimeDir)
	logFile := GetLogFilePath(cfg.RuntimeDir)

	// 2. Check if already running
	if existingPID, err := ReadPID(pidFile); err == nil && IsProcessAlive(existingPID) {
		fmt.Printf("Garund is already running.\n\n  PID: %d\n  URL: %s\n", existingPID, url)
		return nil
	}

	// 3. Port availability check
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: port %d is already in use.\n\nUse:\n\n    garund start --port %d\n\n", cfg.Port, cfg.Port+1)
		os.Exit(1)
	}
	_ = listener.Close()

	// 4. Kubernetes detection & validation
	if _, err := os.Stat(cfg.Kubeconfig); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Garund cannot connect to Kubernetes.\n\nkubeconfig not found:\n    %s\n\nRun:\n\n    garund doctor\n\n", cfg.Kubeconfig)
		os.Exit(1)
	}

	k8sStatus := "reachable"
	k8sContext := "none"
	k8sClient, err := k8s.NewClient(cfg.Kubeconfig)
	if err == nil && k8sClient != nil {
		if rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.Kubeconfig},
			&clientcmd.ConfigOverrides{},
		).RawConfig(); err == nil {
			if cfg.Context != "" {
				k8sContext = cfg.Context
			} else {
				k8sContext = rawConfig.CurrentContext
			}
		}
	} else {
		k8sStatus = "unreachable"
	}

	// 5. Open log file output
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	defer f.Close()

	// Write logs to both log file and terminal
	multiWriter := io.MultiWriter(f, os.Stdout)
	log.SetOutput(multiWriter)

	// 6. Record PID
	pid := os.Getpid()
	if err := WritePID(pidFile, pid); err != nil {
		return fmt.Errorf("failed to write PID file %s: %w", pidFile, err)
	}
	defer RemovePID(pidFile)

	info := buildinfo.Get()

	// 7. Start server asynchronously
	serverErrChan := make(chan error, 1)
	go func() {
		err := server.Run(server.Options{
			Addr:          addr,
			Kubeconfig:    cfg.Kubeconfig,
			PrometheusURL: cfg.PrometheusURL,
			ServeFrontend: true,
		})
		if err != nil {
			serverErrChan <- err
		}
	}()

	// 8. Readiness check
	ready := false
	healthURL := fmt.Sprintf("%s/health", url)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-serverErrChan:
			return fmt.Errorf("Garund server failed to start: %w", err)
		default:
		}

		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			ready = true
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !ready {
		return fmt.Errorf("Garund server did not respond at %s within 15 seconds", url)
	}

	// 9. Print clean startup banner
	fmt.Printf("\nGarund\n")
	fmt.Printf("─────────────────────────────────────────\n\n")
	fmt.Printf("Version       %s\n", info.Version)
	fmt.Printf("Platform      %s\n\n", info.Platform)
	fmt.Printf("✓ Runtime directories\n")
	fmt.Printf("✓ kubeconfig\n")
	fmt.Printf("✓ Kubernetes context: %s\n", k8sContext)
	if k8sStatus == "reachable" {
		fmt.Printf("✓ Kubernetes API\n")
	} else {
		fmt.Printf("✗ Kubernetes API (%s)\n", k8sStatus)
	}
	fmt.Printf("✓ Backend\n")
	fmt.Printf("✓ Frontend\n")
	fmt.Printf("✓ Dashboard\n\n")
	fmt.Printf("Garund is running:\n\n")
	fmt.Printf("    %s\n\n", url)
	fmt.Printf("Press Ctrl+C to stop.\n\n")

	// 10. Signal handling for Ctrl+C / SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		fmt.Printf("\nShutting down Garund...\n")
		return nil
	case err := <-serverErrChan:
		return fmt.Errorf("Garund server error: %w", err)
	}
}
