package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName = "stable.example.com"
	Version   = "v1"
)

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
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
