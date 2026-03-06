package config

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// Spec defines the spec for the config. In order to use below helper methods,
// this must be one of:
// * golang/protobuf Message
// * gogo/protobuf Message
// * Able to marshal/unmarshal using json
type Spec any

type Status any

type Meta struct {
	GroupVersionKind  GroupVersionKind        `json:"type,omitempty"`
	UID               string                  `json:"uid,omitempty"`
	Name              string                  `json:"name,omitempty"`
	Namespace         string                  `json:"namespace,omitempty"`
	Domain            string                  `json:"domain,omitempty"`
	Labels            map[string]string       `json:"labels,omitempty"`
	Annotations       map[string]string       `json:"annotations,omitempty"`
	ResourceVersion   string                  `json:"resourceVersion,omitempty"`
	CreationTimestamp time.Time               `json:"creationTimestamp,omitempty"`
	OwnerReferences   []metav1.OwnerReference `json:"ownerReferences,omitempty"`
	Generation        int64                   `json:"generation,omitempty"`
}

type Config struct {
	Meta

	// Spec holds the configuration object as a gogo protobuf message
	Spec Spec

	// Status holds long-running status.
	Status Status

	// Extra holds additional, non-spec information for internal processing.
	Extra map[string]any
}
