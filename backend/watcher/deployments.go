package watcher

import (
	"context"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WatchDeployments(
	clientset *kubernetes.Clientset,
) {

	watcher, err := clientset.AppsV1().
		Deployments("").
		Watch(
			context.Background(),
			metav1.ListOptions{},
		)

	if err != nil {
		log.Println(err)
		return
	}

	for event := range watcher.ResultChan() {
		deployment := event.Object.(*appsv1.Deployment)

		Broadcast(map[string]any{
			"resource":  "Deployment",
			"action":    string(event.Type),
			"name":      deployment.Name,
			"namespace": deployment.Namespace,
			"ready":     deployment.Status.ReadyReplicas,
		})
	}
}
