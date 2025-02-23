package controller

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"

	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"

	v1 "example.com/controller/pkg/apis/logcleaner/v1"
	corev1 "k8s.io/api/core/v1"
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

func (c Controller) runLogCleanup(logCleaner *v1.LogCleaner) error {
	clientset, err := c.returnClientset()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	pvList, err := c.fetchAssociatedPVC(logCleaner.Spec.VolumeNamePattern)
	if err != nil {
		return fmt.Errorf("failed to fetch PVCs: %w", err)
	}

	log.Println("pvList", pvList)
	log.Println("len(pvList)", len(pvList))

	if len(pvList) == 0 {
		log.Print("No volume names match the regex")
		return nil
	}

	for _, pvc := range pvList {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("log-cleaner-%s-%s", pvc.Name, uuid.NewString()[:8]),
				Namespace: logCleaner.Namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "cleaner",
						Image:   "alpine:latest",
						Command: []string{"sleep", "3600"},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "target-volume",
								MountPath: "/data",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "target-volume",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvc.Name,
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		}

		createdPod, err := clientset.CoreV1().Pods(logCleaner.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create cleaner pod: %w", err)
		}

		err = waitForPodReady(clientset, createdPod.Namespace, createdPod.Name)
		if err != nil {
			return fmt.Errorf("pod failed to become ready: %w", err)
		}

		cmd := []string{
			"sh", "-c",
			"echo 'Listing /data:'; ls -l /data; " +
				"echo 'Running cleanup:'; find /data -type f -mmin +" + fmt.Sprintf("%d", logCleaner.Spec.RetentionPeriod) + " -delete; " +
				"echo 'Cleanup complete'",
		}

		req := clientset.CoreV1().RESTClient().
			Post().
			Namespace(createdPod.Namespace).
			Resource("pods").
			Name(createdPod.Name).
			SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Command:   cmd,
				Container: "cleaner",
				Stdout:    true,
				Stderr:    true,
			}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(&c.config, "POST", req.URL())
		if err != nil {
			return fmt.Errorf("error creating executor: %w", err)
		}

		var stdout, stderr bytes.Buffer
		err = exec.Stream(remotecommand.StreamOptions{
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			return fmt.Errorf("error executing cleanup command: %w\nStderr: %s", err, stderr.String())
		}
		log.Printf("Cleanup command output:\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())

		err = clientset.CoreV1().Pods(createdPod.Namespace).Delete(
			context.Background(),
			createdPod.Name,
			metav1.DeleteOptions{},
		)
		if err != nil {
			log.Printf("Warning: failed to delete cleaner pod %s: %v", createdPod.Name, err)
		}
	}

	return nil
}

func waitForPodReady(clientset *kubernetes.Clientset, namespace, name string) error {
	return wait.PollImmediate(time.Second, time.Minute*2, func() (bool, error) {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
}

func (c *Controller) fetchAssociatedPVC(volumeNamePattern string) ([]corev1.PersistentVolumeClaim, error) {
	if c.coreRestClient == nil {
		return nil, fmt.Errorf("coreRestClient is not initialized")
	}

	var allPVCs corev1.PersistentVolumeClaimList
	if err := c.coreRestClient.
		Get().
		Resource("persistentvolumeclaims").
		Do(context.TODO()).
		Into(&allPVCs); err != nil {
		log.Printf("Error fetching PVs: %v", err)
		return nil, err
	}

	pattern, err := regexp.Compile(volumeNamePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid volume name pattern '%s': %w", volumeNamePattern, err)
	}

	matchedPVCs := make([]corev1.PersistentVolumeClaim, 0, len(allPVCs.Items))

	for _, pvc := range allPVCs.Items {
		if pattern.MatchString(pvc.Name) {
			matchedPVCs = append(matchedPVCs, pvc)
		}
	}

	return matchedPVCs, nil
}
