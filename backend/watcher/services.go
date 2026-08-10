package watcher

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

func WatchServices(
	clientset *kubernetes.Clientset,
) {

	watcher, err := clientset.CoreV1().
		Services("").
		Watch(
			context.Background(),
			metav1.ListOptions{},
		)

	if err != nil {
		log.Println(err)
		return
	}

	for event := range watcher.ResultChan() {

		svc := event.Object.(*corev1.Service)

		log.Printf(
			"SERVICE WATCH: %s %s/%s",
			event.Type,
			svc.Namespace,
			svc.Name,
		)

		Broadcast(map[string]any{
			"resource":  "Service",
			"action":    string(event.Type),
			"name":      svc.Name,
			"namespace": svc.Namespace,
		})

	}

}
