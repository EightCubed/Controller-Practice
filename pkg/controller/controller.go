package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	v1 "example.com/controller/pkg/apis/logcleaner/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const DEFAULT_CLEAN_UP_INTERVAL = 24

type Controller struct {
	config         rest.Config
	coreRestClient rest.Interface
	restClient     rest.Interface
}

func NewController(config rest.Config, coreRestClient rest.Interface, restClient rest.Interface) *Controller {
	return &Controller{
		config:         config,
		restClient:     restClient,
		coreRestClient: coreRestClient,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	cleanUpInterval := DEFAULT_CLEAN_UP_INTERVAL

	if value, found := os.LookupEnv("CLEAN_UP_INTERVAL_IN_HOURS"); found {
		if parsedValue, err := strconv.Atoi(value); err == nil {
			cleanUpInterval = parsedValue
		} else {
			log.Printf("Invalid CLEAN_UP_INTERVAL_IN_HOURS value: %s, using default: %d hours", value, cleanUpInterval)
		}
	}

	ticketFireDuration := time.Duration(cleanUpInterval) * time.Hour
	ticker := time.NewTicker(ticketFireDuration)
	defer ticker.Stop()

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

	go controller.Run(stopCh)

	for {
		select {
		case <-ticker.C:
			log.Println("24-hour ticker fired. Running cleanup...")
			c.runPeriodicCleanup()
		case <-stopCh:
			log.Println("Stopping controller...")
			return
		}
	}
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

	err := c.runLogCleanup(logCleaner)
	if err != nil {
		log.Printf("Error : %v\n", err)
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

	err := c.runLogCleanup(newLogCleaner)
	if err != nil {
		log.Printf("Error : %v\n", err)
	}
}

func (c *Controller) onDelete(obj interface{}) {
	logCleaner, ok := obj.(*v1.LogCleaner)
	if !ok {
		log.Printf("Error: unexpected type %T", obj)
		return
	}
	fmt.Printf("LogCleaner deleted: %s in namespace %s\n", logCleaner.Name, logCleaner.Namespace)
}

func (c *Controller) runPeriodicCleanup() {
	var logCleaners v1.LogCleanerList
	err := c.restClient.Get().
		Namespace(metav1.NamespaceAll).
		Resource("logcleaners").
		Do(context.Background()).
		Into(&logCleaners)
	if err != nil {
		log.Printf("Error fetching LogCleaner resources: %v\n", err)
		return
	}

	for _, logCleaner := range logCleaners.Items {
		err := c.runLogCleanup(&logCleaner)
		if err != nil {
			log.Printf("Error running cleanup for LogCleaner %s: %v\n", logCleaner.Name, err)
		}
	}
}
