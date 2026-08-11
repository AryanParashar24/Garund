package server

import (
	"testing"
	"time"

	"github.com/garund/garund/internal/agent"
	k8s "github.com/garund/garund/internal/kubernetes"
)

func TestMultiClusterManagerRegistration(t *testing.T) {
	cm := k8s.NewClusterManager()

	c1 := &k8s.ClusterConnection{
		ID:                "cluster-prod-01",
		Name:              "Production EKS",
		Environment:       "production",
		Provider:          "AWS / EKS",
		ClusterType:       "EKS",
		ConnectionMode:    k8s.ModeAgent,
		Status:            k8s.StatusConnected,
		KubernetesVersion: "v1.32.0",
		Capabilities: k8s.CapabilitySet{
			CanReadWorkloads:    true,
			CanReadLogs:         true,
			CanReadEvents:       true,
			CanReadTelemetry:    true,
			CanOperateWorkloads: false,
			CanAdminister:       false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	c2 := &k8s.ClusterConnection{
		ID:                "cluster-staging-01",
		Name:              "Staging GKE",
		Environment:       "staging",
		Provider:          "GCP / GKE",
		ClusterType:       "GKE",
		ConnectionMode:    k8s.ModeServiceAccountToken,
		Status:            k8s.StatusConnected,
		KubernetesVersion: "v1.31.2",
		Capabilities: k8s.CapabilitySet{
			CanReadWorkloads:    true,
			CanReadLogs:         true,
			CanReadEvents:       true,
			CanReadTelemetry:    true,
			CanOperateWorkloads: true,
			CanAdminister:       true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	cm.RegisterCluster(c1, nil)
	cm.RegisterCluster(c2, nil)

	clusters := cm.ListClusters()
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters registered, got %d", len(clusters))
	}

	// Default active cluster should be first registered
	if cm.GetActiveClusterID() != "cluster-prod-01" {
		t.Fatalf("expected active cluster 'cluster-prod-01', got '%s'", cm.GetActiveClusterID())
	}

	// Test active switching
	if err := cm.SetActiveCluster("cluster-staging-01"); err != nil {
		t.Fatalf("failed to set active cluster: %v", err)
	}
	if cm.GetActiveClusterID() != "cluster-staging-01" {
		t.Fatalf("expected active cluster 'cluster-staging-01', got '%s'", cm.GetActiveClusterID())
	}

	// Verify RBAC capability isolation
	retrievedC1, found := cm.GetCluster("cluster-prod-01")
	if !found {
		t.Fatalf("cluster-prod-01 not found")
	}
	if retrievedC1.Capabilities.CanOperateWorkloads {
		t.Fatalf("expected CanOperateWorkloads to be false for read-only prod cluster")
	}
}

func TestAgentRegistryTokenValidation(t *testing.T) {
	registry := agent.GetAgentRegistry()
	clusterID := "cluster-agent-test"
	token := "gtok_test_12345"

	registry.RegisterToken(clusterID, token)

	if !registry.ValidateToken(clusterID, token) {
		t.Fatalf("expected valid token to pass validation")
	}

	if registry.ValidateToken(clusterID, "wrong-token") {
		t.Fatalf("expected invalid token to fail validation")
	}
}
