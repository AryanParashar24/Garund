package watcher

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WatchPods(
	clientset *kubernetes.Clientset,
) {

	watcher, err := clientset.CoreV1().
		Pods("").
		Watch(
			context.Background(),
			metav1.ListOptions{},
		)

	if err != nil {
		log.Println(err)
		return
	}

	for event := range watcher.ResultChan() {

		pod := event.Object.(*corev1.Pod)

		log.Printf(
			"POD WATCH: %s %s/%s",
			event.Type,
			pod.Namespace,
			pod.Name,
		)

		Broadcast(map[string]any{
			"resource":  "Pod",
			"action":    string(event.Type),
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"phase":     string(pod.Status.Phase),
		})
	}

}
