package main

import (
	"log"
	"os"

	"github.com/garund/garund/internal/agent"
	"github.com/garund/garund/internal/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "agent" {
		serverURL := os.Getenv("GARUND_SERVER_URL")
		clusterID := os.Getenv("GARUND_CLUSTER_ID")
		agentToken := os.Getenv("GARUND_AGENT_TOKEN")
		kubeconfig := os.Getenv("KUBECONFIG")

		if err := agent.Run(agent.AgentOptions{
			ServerURL:       serverURL,
			ClusterID:       clusterID,
			EnrollmentToken: agentToken,
			Kubeconfig:      kubeconfig,
		}); err != nil {
			log.Fatalf("Garund Agent error: %v", err)
		}
		return
	}

	// Default: Run control plane server
	if err := server.Run(server.Options{
		Addr:          os.Getenv("GARUND_ADDR"),
		Kubeconfig:    os.Getenv("KUBECONFIG"),
		PrometheusURL: os.Getenv("PROMETHEUS_URL"),
	}); err != nil {
		log.Fatal(err)
	}
}
