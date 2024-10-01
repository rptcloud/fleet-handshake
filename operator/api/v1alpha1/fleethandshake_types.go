package v1alpha1

import (
	fleetv1alpha1api "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make generate" to regenerate code after modifying this file
type FleetHandshakeSpec struct {
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:description="The name of the secret in which you want to sync to the target clusters"
	SecretName string `json:"secretName"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:description="The namespace of the secret in which you want to sync to the target clusters"
	SecretNamespace string `json:"secretNamespace"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:description="The namespace to sync the secret to in the target clusters"
	TargetNamespace string `json:"targetNamespace"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinSize=1
	//+kubebuilder:description="The targets to sync the secret to"
	Targets []fleetv1alpha1api.BundleTarget `json:"targets"`
}

// Important: Run "make generate" to regenerate code after modifying this file
type FleetHandshakeStatus struct {
	//+kubebuilder:validation:Optional
	//+kubebuilder:description="The status of the handshake. Can be 'Pending', 'Missing', 'Synced', 'Error'"
	//+kubebuilder:default="Pending"
	Status string `json:"status"`
}

//+kubebuilder:object:root=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:subresource:status

// FleetHandshake is the Schema for the fleethandshakes API
type FleetHandshake struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetHandshakeSpec   `json:"spec,omitempty"`
	Status FleetHandshakeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetHandshakeList contains a list of FleetHandshake
type FleetHandshakeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FleetHandshake `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FleetHandshake{}, &FleetHandshakeList{})
}
