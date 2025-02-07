package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// LogCleaner represents the custom resource
type LogCleaner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LogCleanerSpec `json:"spec,omitempty"`
}

// LogCleanerSpec defines the desired state of LogCleaner
type LogCleanerSpec struct {
	RetentionPeriod   int    `json:"retentionPeriod"`
	TargetNamespace   string `json:"targetNamespace"`
	VolumeNamePattern string `json:"volumeNamePattern"`
}

// LogCleanerList is a list of LogCleaner resources
type LogCleanerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LogCleaner `json:"items"`
}

// DeepCopyObject implements runtime.Object for LogCleaner
func (in *LogCleaner) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(LogCleaner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into the given LogCleaner
func (in *LogCleaner) DeepCopyInto(out *LogCleaner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

// DeepCopyObject implements runtime.Object for LogCleanerList
func (in *LogCleanerList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(LogCleanerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into the given LogCleanerList
func (in *LogCleanerList) DeepCopyInto(out *LogCleanerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LogCleaner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// GroupVersion for the custom resource
var GroupVersion = schema.GroupVersion{Group: "stable.example.com", Version: "v1"}

func main() {
	// Create an in-cluster Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("❌ Error creating in-cluster config: %v", err)
	}

	// Create a new scheme and register the custom resource
	scheme := runtime.NewScheme()
	AddToScheme(scheme)

	// Configure the REST client
	config.APIPath = "/apis"
	config.GroupVersion = &GroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalf("❌ Error creating REST client: %v", err)
	}

	// Create a stop channel for graceful shutdown
	stopCh := make(chan struct{})
	go watchCRD(restClient, stopCh)

	// Set up signal handling for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a termination signal
	log.Println("✅ Watching for LogCleaner events...")
	<-signalCh

	// Stop the informer
	close(stopCh)
	log.Println("🛑 Shutting down...")
}

// Function to watch for LogCleaner events
func watchCRD(restClient rest.Interface, stopCh <-chan struct{}) {
	// Create a ListWatch for LogCleaners
	watchList := cache.NewListWatchFromClient(
		restClient,
		"logcleaners", // Plural name of the custom resource
		"",            // Namespace (empty string for all namespaces)
		fields.Everything(),
	)

	// Define event handlers
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    onLogCleanerAdd,
		UpdateFunc: onLogCleanerUpdate,
		DeleteFunc: onLogCleanerDelete,
	}

	// Create an informer
	_, controller := cache.NewInformer(
		watchList,
		&LogCleaner{}, // The type of object to watch
		0,             // Resync period (0 to disable resync)
		handlers,      // Event handlers
	)

	// Start the controller
	controller.Run(stopCh)
}

// Event Handlers for LogCleaner
func onLogCleanerAdd(obj interface{}) {
	logCleaner := obj.(*LogCleaner)
	fmt.Printf("📢 LogCleaner added: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
}

func onLogCleanerUpdate(oldObj, newObj interface{}) {
	oldLogCleaner := oldObj.(*LogCleaner)
	newLogCleaner := newObj.(*LogCleaner)
	fmt.Printf("📢 LogCleaner updated: %s (RetentionPeriod: %d -> %d)\n", newLogCleaner.Name, oldLogCleaner.Spec.RetentionPeriod, newLogCleaner.Spec.RetentionPeriod)
}

func onLogCleanerDelete(obj interface{}) {
	logCleaner := obj.(*LogCleaner)
	fmt.Printf("📢 LogCleaner deleted: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
}

// AddToScheme registers the custom resource with the scheme
func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &LogCleaner{}, &LogCleanerList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
