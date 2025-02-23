package controller

import (
	"fmt"
	"log"

	v1 "example.com/controller/pkg/apis/logcleaner/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	config         rest.Config
	coreRestClient rest.Interface
	restClient     rest.Interface
}

func NewController(config rest.Config, coreRestClient rest.Interface, restClient rest.Interface) *Controller {
	return &Controller{
		config:     config,
		restClient: restClient,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	watchList := cache.NewListWatchFromClient(
		c.restClient,
		"logcleaners",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		watchList,
		&v1.LogCleaner{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "LogCleaner",
			},
		},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)

	controller.Run(stopCh)
}

func (c *Controller) onAdd(obj interface{}) {
	logCleaner, ok := obj.(*v1.LogCleaner)
	if !ok {
		log.Printf("Error: unexpected type %T", obj)
		return
	}
	fmt.Printf("LogCleaner added: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
	fmt.Printf("RetentionPeriod: %d, TargetNamespace: %s, VolumeNamePattern: %s\n",
		logCleaner.Spec.RetentionPeriod,
		logCleaner.Spec.TargetNamespace,
		logCleaner.Spec.VolumeNamePattern)

	err := c.fetchAssociatedPVC()
	if err != nil {
		fmt.Printf("Error : %v\n", err)
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	oldLogCleaner, ok1 := oldObj.(*v1.LogCleaner)
	newLogCleaner, ok2 := newObj.(*v1.LogCleaner)
	if !ok1 || !ok2 {
		log.Printf("Error: unexpected types %T and %T", oldObj, newObj)
		return
	}

	fmt.Printf("LogCleaner updated: %s in namespace %s\n", newLogCleaner.Name, newLogCleaner.Namespace)
	fmt.Printf("RetentionPeriod: %d -> %d\n",
		oldLogCleaner.Spec.RetentionPeriod,
		newLogCleaner.Spec.RetentionPeriod)
}

func (c *Controller) onDelete(obj interface{}) {
	logCleaner, ok := obj.(*v1.LogCleaner)
	if !ok {
		log.Printf("Error: unexpected type %T", obj)
		return
	}
	fmt.Printf("LogCleaner deleted: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
}
