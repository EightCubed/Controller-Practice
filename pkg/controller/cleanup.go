package controller

import (
	"bytes"
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
)

func (c Controller) returnClientset() (clientset *kubernetes.Clientset, err error) {
	clientset, err = kubernetes.NewForConfig(&c.config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %w", err)
	}

	return clientset, nil
}

func (c Controller) runLogCleanup(logFilePath string, containerName string, podName string, namespace string, retentionDays int) error {
	clientset, err := c.returnClientset()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	cmd := []string{"find", logFilePath, "-type", "f", "-mtime", fmt.Sprintf("+%d", retentionDays), "-delete"}

	req := clientset.CoreV1().RESTClient().
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   cmd,
			Container: containerName,
			Stdout:    true,
			Stderr:    true,
		}, metav1.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(&c.config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("error executing cleanup command: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("error executing cleanup command: %w\nStderr: %s", err, stderr.String())
	}

	log.Printf("✅ Deleted logs:\n%s", stdout.String())

	return nil
}

func (c Controller) fetchAssociatedPVC() error {
	var pvList v1.PersistentVolumeList
	err := c.coreRestClient.
		Get().
		Resource("persistentvolumes").
		Do(context.TODO()).
		Into(&pvList)
	if err != nil {
		return err
	}

	fmt.Printf("pvList : %v", pvList)

	return nil
}
