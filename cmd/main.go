package main

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// Create an in-cluster Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error creating in-cluster config: %v", err)
	}

	// fmt.Printf("%v", config)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	// Watch for Pod events
	watcher, err := clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error watching Pods: %v", err)
	}

	fmt.Printf("%v", watcher)
}
