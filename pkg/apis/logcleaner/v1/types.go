package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogCleaner defines the schema for the logcleaner custom resource
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

// LogCleanerList contains a list of LogCleaner resources
type LogCleanerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LogCleaner `json:"items"`
}
