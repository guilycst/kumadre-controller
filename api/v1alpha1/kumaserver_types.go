package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KumaServerAuth struct {
	UsernamePassword *UsernamePasswordAuth `json:"usernamePassword,omitempty"`
	APIToken         *APITokenAuth         `json:"apiToken,omitempty"`
	OIDC             *OIDCAuth             `json:"oidc,omitempty"`
}

type KumaServerSpec struct {
	URL                string         `json:"url"`
	Auth               KumaServerAuth `json:"auth"`
	Default            bool           `json:"default,omitempty"`
	InsecureSkipVerify bool           `json:"insecureSkipVerify,omitempty"`
	Timeout            string         `json:"timeout,omitempty"`
}

type KumaServerStatus struct {
	Connected   bool   `json:"connected"`
	LastChecked string `json:"lastChecked"`
	Error       string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kumaservers,scope=Namespaced,shortName=kumasrv
// +kubebuilder:groupName=kumadre.controller
type KumaServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KumaServerSpec   `json:"spec,omitempty"`
	Status            KumaServerStatus `json:"status,omitempty"`
}

type KumaServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KumaServer `json:"items"`
}
