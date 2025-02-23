package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	v1 "example.com/controller/pkg/apis/logcleaner/v1"
	"example.com/controller/pkg/controller"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("❌ Error creating in-cluster config: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		log.Fatalf("❌ Error adding types to scheme: %v", err)
	}

	config.APIPath = "/apis"
	config.GroupVersion = &v1.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	config.ContentType = runtime.ContentTypeJSON

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalf("❌ Error creating REST client: %v", err)
	}

	stopCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	controller := controller.NewController(restClient)
	go controller.Run(stopCh)

	log.Println("✅ Watching for LogCleaner events...")
	<-signalCh

	close(stopCh)
	log.Println("🛑 Shutting down...")
}
