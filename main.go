package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/garund/garund/internal/agent"
	"github.com/garund/garund/internal/cli"
	"github.com/garund/garund/internal/server"
)

func main() {
	if len(os.Args) > 1 {
		sub := os.Args[1]
		switch sub {
		case "agent":
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

		case "start":
			fs := flag.NewFlagSet("start", flag.ExitOnError)
			host := fs.String("host", "127.0.0.1", "Host bind address")
			port := fs.Int("port", 8080, "Port number")
			kubeconfig := fs.String("kubeconfig", "", "Path to kubeconfig file")
			contextName := fs.String("context", "", "Kubernetes context name")
			promURL := fs.String("prometheus-url", "", "Prometheus server URL")
			_ = fs.Parse(os.Args[2:])

			err := cli.RunStart(cli.StartOptions{
				Host:          *host,
				Port:          *port,
				Kubeconfig:    *kubeconfig,
				Context:       *contextName,
				PrometheusURL: *promURL,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error starting Garund: %v\n", err)
				os.Exit(1)
			}
			return

		case "stop":
			if err := cli.RunStop(); err != nil {
				fmt.Fprintf(os.Stderr, "Error stopping Garund: %v\n", err)
				os.Exit(1)
			}
			return

		case "restart":
			fs := flag.NewFlagSet("restart", flag.ExitOnError)
			host := fs.String("host", "127.0.0.1", "Host bind address")
			port := fs.Int("port", 8080, "Port number")
			kubeconfig := fs.String("kubeconfig", "", "Path to kubeconfig file")
			contextName := fs.String("context", "", "Kubernetes context name")
			promURL := fs.String("prometheus-url", "", "Prometheus server URL")
			_ = fs.Parse(os.Args[2:])

			err := cli.RunRestart(cli.StartOptions{
				Host:          *host,
				Port:          *port,
				Kubeconfig:    *kubeconfig,
				Context:       *contextName,
				PrometheusURL: *promURL,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error restarting Garund: %v\n", err)
				os.Exit(1)
			}
			return

		case "status":
			cli.RunStatus()
			return

		case "logs":
			if err := cli.RunLogs(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading logs: %v\n", err)
				os.Exit(1)
			}
			return

		case "doctor":
			cli.RunDoctor()
			return

		case "version":
			cli.RunVersion()
			return

		case "help", "-h", "--help":
			printUsage()
			return
		}
	}

	// Default fallback (e.g. go run . with no args in development): run server directly
	addr := os.Getenv("GARUND_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	if err := server.Run(server.Options{
		Addr:          addr,
		Kubeconfig:    os.Getenv("KUBECONFIG"),
		PrometheusURL: os.Getenv("PROMETHEUS_URL"),
		ServeFrontend: true,
	}); err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Printf(`Garund — Kubernetes SRE Observability & Reliability Control Plane

Usage:
    garund <command> [flags]

Commands:
    start       Start Garund server and dashboard
    stop        Stop running Garund server
    restart     Restart Garund server
    status      Show status of Garund process and Kubernetes connection
    logs        Display Garund server logs
    doctor      Diagnose installation, permissions, ports, and Kubernetes API
    version     Show Garund version and build information
    help        Show command help

Flags for 'start' / 'restart':
    --host           Host address to bind (default: 127.0.0.1)
    --port           Port number to listen on (default: 8080)
    --kubeconfig     Path to kubeconfig file (default: ~/.kube/config)
    --context        Kubernetes context name
    --prometheus-url Custom Prometheus URL

Examples:
    garund start
    garund start --port 8081
    garund doctor
    garund status
    garund stop
`)
}
