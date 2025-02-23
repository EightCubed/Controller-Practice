package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
