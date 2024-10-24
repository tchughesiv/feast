/*
Copyright 2024 Feast Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Feast phases:
	ReadyPhase   = "Ready"
	PendingPhase = "Pending"
	FailedPhase  = "Failed"

	// Feast condition types:
	ClientReadyType       = "Client"
	OfflineStoreReadyType = "OfflineStore"
	OnlineStoreReadyType  = "OnlineStore"
	RegistryReadyType     = "Registry"
	ReadyType             = "FeatureStore"

	// Feast condition reasons:
	ReadyReason              = "Ready"
	FailedReason             = "FeatureStoreFailed"
	OfflineStoreFailedReason = "OfflineStoreDeploymentFailed"
	OnlineStoreFailedReason  = "OnlineStoreDeploymentFailed"
	RegistryFailedReason     = "RegistryDeploymentFailed"
	ClientFailedReason       = "ClientDeploymentFailed"

	// Feast condition messages:
	ReadyMessage             = "FeatureStore installation complete"
	OfflineStoreReadyMessage = "Offline Store installation complete"
	OnlineStoreReadyMessage  = "Online Store installation complete"
	RegistryReadyMessage     = "Registry installation complete"
	ClientReadyMessage       = "Client installation complete"

	// entity_key_serialization_version
	SerializationVersion = 3
)

// FeatureStoreSpec defines the desired state of FeatureStore
type FeatureStoreSpec struct {
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9][A-Za-z0-9_]*$"
	// FeastProject is the Feast project id. This can be any alphanumeric string with underscores, but it cannot start with an underscore. Required.
	FeastProject string                `json:"feastProject"`
	Services     *FeatureStoreServices `json:"services,omitempty"`
}

// FeatureStoreServices defines the desired feast services. ephemeral registry is deployed by default.
type FeatureStoreServices struct {
	OfflineStore *OfflineStore `json:"offlineStore,omitempty"`
	OnlineStore  *OnlineStore  `json:"onlineStore,omitempty"`
	Registry     *Registry     `json:"registry,omitempty"`
}

// OfflineStore
type OfflineStore struct {
	ServiceConfig `json:",inline"`
}

// OnlineStore
type OnlineStore struct {
	ServiceConfig `json:",inline"`
}

// Registry
type Registry struct {
	ServiceConfig `json:",inline"`
}

// ServiceConfig
type ServiceConfig struct {
	Image            *string `json:"image,omitempty"`
	OptServiceConfig `json:",inline"`
}

// OptServiceConfig
type OptServiceConfig struct {
	ImagePullPolicy *corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// FeatureStoreStatus defines the observed state of FeatureStore
type FeatureStoreStatus struct {
	Applied          FeatureStoreSpec   `json:"applied,omitempty"`
	ClientConfigMap  string             `json:"clientConfigMap,omitempty"`
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
	FeastVersion     string             `json:"feastVersion,omitempty"`
	Phase            string             `json:"phase,omitempty"`
	ServiceHostnames ServiceHostnames   `json:"serviceHostnames,omitempty"`
}

// ServiceHostnames
type ServiceHostnames struct {
	OfflineStore string `json:"offlineStore,omitempty"`
	OnlineStore  string `json:"onlineStore,omitempty"`
	Registry     string `json:"registry,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=feast
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// FeatureStore is the Schema for the featurestores API
type FeatureStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FeatureStoreSpec   `json:"spec,omitempty"`
	Status FeatureStoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FeatureStoreList contains a list of FeatureStore
type FeatureStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FeatureStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FeatureStore{}, &FeatureStoreList{})
}
