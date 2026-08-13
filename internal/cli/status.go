package cli

import (
	"fmt"
	"net/http"
	"time"

	k8s "github.com/garund/garund/internal/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func RunStatus() {
	cfg := DefaultConfig()
	pidFile := GetPIDFilePath(cfg.RuntimeDir)
	addr := GetAddr(cfg.Host, cfg.Port)
	url := fmt.Sprintf("http://%s", addr)

	fmt.Printf("\nGarund Status\n")
	fmt.Printf("─────────────────────────\n")

	pid, err := ReadPID(pidFile)
	if err != nil || !IsProcessAlive(pid) {
		if err == nil {
			_ = RemovePID(pidFile)
		}
		fmt.Printf("Status:       stopped\n\n")
		return
	}

	// Check HTTP health
	statusStr := "running (unresponsive)"
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/health", url))
	if err == nil && resp.StatusCode == http.StatusOK {
		statusStr = "running"
		_ = resp.Body.Close()
	}

	k8sStatus := "disconnected"
	k8sContext := "none"
	if k8sClient, err := k8s.NewClient(cfg.Kubeconfig); err == nil && k8sClient != nil {
		k8sStatus = "connected"
		if rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.Kubeconfig},
			&clientcmd.ConfigOverrides{},
		).RawConfig(); err == nil {
			k8sContext = rawConfig.CurrentContext
		}
	}

	fmt.Printf("Status:       %s\n", statusStr)
	fmt.Printf("PID:          %d\n", pid)
	fmt.Printf("Address:      %s\n", addr)
	fmt.Printf("Kubernetes:   %s\n", k8sStatus)
	fmt.Printf("Context:      %s\n\n", k8sContext)
}
