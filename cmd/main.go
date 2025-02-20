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

var SchemeGroupVersion = schema.GroupVersion{Group: "stable.example.com", Version: "v1"}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

type LogCleaner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LogCleanerSpec `json:"spec,omitempty"`
}

type LogCleanerSpec struct {
	RetentionPeriod   int    `json:"retentionPeriod"`
	TargetNamespace   string `json:"targetNamespace"`
	VolumeNamePattern string `json:"volumeNamePattern"`
}

type LogCleanerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LogCleaner `json:"items"`
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&LogCleaner{},
		&LogCleanerList{},
	)

	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{
			Group:   SchemeGroupVersion.Group,
			Version: runtime.APIVersionInternal,
			Kind:    "LogCleaner",
		},
		&LogCleaner{},
	)

	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{
			Group:   SchemeGroupVersion.Group,
			Version: runtime.APIVersionInternal,
			Kind:    "LogCleanerList",
		},
		&LogCleanerList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func (in *LogCleaner) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(LogCleaner)
	in.DeepCopyInto(out)
	return out
}

func (in *LogCleaner) DeepCopyInto(out *LogCleaner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *LogCleanerList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(LogCleanerList)
	in.DeepCopyInto(out)
	return out
}

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

func (in *LogCleaner) GetObjectKind() schema.ObjectKind {
	return &in.TypeMeta
}

func (in *LogCleanerList) GetObjectKind() schema.ObjectKind {
	return &in.TypeMeta
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("❌ Error creating in-cluster config: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		log.Fatalf("❌ Error adding types to scheme: %v", err)
	}

	config.APIPath = "/apis"
	config.GroupVersion = &SchemeGroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	config.ContentType = runtime.ContentTypeJSON

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalf("❌ Error creating REST client: %v", err)
	}

	stopCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go watchCRD(restClient, stopCh)

	log.Println("✅ Watching for LogCleaner events...")
	<-signalCh

	close(stopCh)
	log.Println("🛑 Shutting down...")
}

func watchCRD(restClient rest.Interface, stopCh <-chan struct{}) {
	watchList := cache.NewListWatchFromClient(
		restClient,
		"logcleaners",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    onLogCleanerAdd,
		UpdateFunc: onLogCleanerUpdate,
		DeleteFunc: onLogCleanerDelete,
	}

	_, controller := cache.NewInformer(
		watchList,
		&LogCleaner{
			TypeMeta: metav1.TypeMeta{
				APIVersion: SchemeGroupVersion.String(),
				Kind:       "LogCleaner",
			},
		},
		0,
		handlers,
	)

	controller.Run(stopCh)
}

func onLogCleanerAdd(obj interface{}) {
	logCleaner, ok := obj.(*LogCleaner)
	if !ok {
		log.Printf("❌ Error: unexpected type %T", obj)
		return
	}
	fmt.Printf("📢 LogCleaner added: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
	fmt.Printf("   RetentionPeriod: %d, TargetNamespace: %s, VolumeNamePattern: %s\n",
		logCleaner.Spec.RetentionPeriod,
		logCleaner.Spec.TargetNamespace,
		logCleaner.Spec.VolumeNamePattern)
}

func onLogCleanerUpdate(oldObj, newObj interface{}) {
	oldLogCleaner, ok1 := oldObj.(*LogCleaner)
	newLogCleaner, ok2 := newObj.(*LogCleaner)
	if !ok1 || !ok2 {
		log.Printf("❌ Error: unexpected types %T and %T", oldObj, newObj)
		return
	}
	fmt.Printf("📢 LogCleaner updated: %s in namespace %s\n", newLogCleaner.Name, newLogCleaner.Namespace)
	fmt.Printf("   RetentionPeriod: %d -> %d\n",
		oldLogCleaner.Spec.RetentionPeriod,
		newLogCleaner.Spec.RetentionPeriod)
}

func onLogCleanerDelete(obj interface{}) {
	logCleaner, ok := obj.(*LogCleaner)
	if !ok {
		log.Printf("❌ Error: unexpected type %T", obj)
		return
	}
	fmt.Printf("📢 LogCleaner deleted: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
}
