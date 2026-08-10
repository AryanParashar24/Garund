package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientInfo holds Kubernetes connection details.
type ClientInfo struct {
	KubeconfigPath string
	CurrentContext string
	Clientset      *kubernetes.Clientset
}

// DefaultKubeconfigPath returns ~/.kube/config.
func DefaultKubeconfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "config"), nil
}

// ResolveKubeconfig returns the kubeconfig path from env or default location.
func ResolveKubeconfig() (string, error) {
	if path := os.Getenv("KUBECONFIG"); path != "" {
		return path, nil
	}
	return DefaultKubeconfigPath()
}

// NewClient builds a Kubernetes clientset from the given kubeconfig path.
func NewClient(kubeconfigPath string) (*ClientInfo, error) {
	if kubeconfigPath == "" {
		var err error
		kubeconfigPath, err = ResolveKubeconfig()
		if err != nil {
			return nil, err
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig %s: %w", kubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	rawConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}

	return &ClientInfo{
		KubeconfigPath: kubeconfigPath,
		CurrentContext: rawConfig.CurrentContext,
		Clientset:      clientset,
	}, nil
}

// CanListPods verifies read access to pods cluster-wide.
func CanListPods(info *ClientInfo) error {
	_, err := info.Clientset.CoreV1().Pods("").List(
		context.Background(),
		metav1.ListOptions{Limit: 1},
	)
	return err
}

// CanListServices verifies read access to services.
func CanListServices(info *ClientInfo) error {
	_, err := info.Clientset.CoreV1().Services("").List(
		context.Background(),
		metav1.ListOptions{Limit: 1},
	)
	return err
}

// CanListEvents verifies read access to events.
func CanListEvents(info *ClientInfo) error {
	_, err := info.Clientset.CoreV1().Events("").List(
		context.Background(),
		metav1.ListOptions{Limit: 1},
	)
	return err
}
