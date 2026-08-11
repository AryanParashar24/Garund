package main

import (
	"log"
	"os"

	"github.com/garund/garund/internal/agent"
)

func main() {
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
}
