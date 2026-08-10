package actions

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

func DeletePod(
	client *kubernetes.Clientset,
	namespace string,
	name string,
) error {

	return client.CoreV1().
		Pods(namespace).
		Delete(
			context.Background(),
			name,
			metav1.DeleteOptions{},
		)
}

func RestartPod(
	client *kubernetes.Clientset,
	namespace string,
	name string,
) error {

	// Kubernetes recreates it automatically
	return DeletePod(client, namespace, name)
}

func DescribePod(
	client *kubernetes.Clientset,
	namespace string,
	name string,
) (*corev1.Pod, error) {

	return client.CoreV1().
		Pods(namespace).
		Get(
			context.Background(),
			name,
			metav1.GetOptions{},
		)
}
