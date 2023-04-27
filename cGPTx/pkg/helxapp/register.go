package helxapp

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "renci.org", Version: "v1"}

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addToScheme is a global function that registers this API group & version to a scheme
func AddToScheme(scheme *runtime.Scheme) error {
	schemeBuilder := runtime.NewSchemeBuilder(AddKnownTypes)
	return schemeBuilder.AddToScheme(scheme)
}

// AddKnownTypes adds the set of types defined in this package to the supplied scheme.
func AddKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&HeLxApp{},
		&HeLxAppList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
