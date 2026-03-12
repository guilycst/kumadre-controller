package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UptimeKumaAuth struct {
	UsernamePassword *UsernamePasswordAuth `json:"usernamePassword,omitempty"`
	APIToken         *APITokenAuth         `json:"apiToken,omitempty"`
	OIDC             *OIDCAuth             `json:"oidc,omitempty"`
}

type UsernamePasswordAuth struct {
	Username          string    `json:"username"`
	PasswordSecretRef SecretRef `json:"passwordSecretRef"`
}

type APITokenAuth struct {
	TokenSecretRef SecretRef `json:"tokenSecretRef"`
}

type OIDCAuth struct {
	ClientID              string    `json:"clientId"`
	ClientSecretSecretRef SecretRef `json:"clientSecretSecretRef"`
	TokenURL              string    `json:"tokenUrl"`
	Scopes                []string  `json:"scopes,omitempty"`
}

type SecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type UptimeKumaConfig struct {
	URL                string         `json:"url"`
	Auth               UptimeKumaAuth `json:"auth"`
	InsecureSkipVerify bool           `json:"insecureSkipVerify,omitempty"`
	Timeout            string         `json:"timeout,omitempty"`
}

type DiscoveryMode string

const (
	DiscoveryModeLazy  DiscoveryMode = "lazy"
	DiscoveryModeEager DiscoveryMode = "eager"
)

type ResourceKind string

const (
	ResourceKindIngress   ResourceKind = "Ingress"
	ResourceKindHTTPRoute ResourceKind = "HTTPRoute"
	ResourceKindTCPRoute  ResourceKind = "TCPRoute"
	ResourceKindService   ResourceKind = "Service"
	ResourceKindEndpoint  ResourceKind = "Endpoint"
	ResourceKindNode      ResourceKind = "Node"
)

type MonitorType string

const (
	MonitorTypeHTTP  MonitorType = "http"
	MonitorTypeHTTPS MonitorType = "https"
	MonitorTypePing  MonitorType = "ping"
)

type MonitorConfig struct {
	Type       MonitorType `json:"type"`
	Interval   int         `json:"interval,omitempty"`
	Timeout    int         `json:"timeout,omitempty"`
	MaxRetries int         `json:"maxRetries,omitempty"`
}

type KumaMonitorSpec struct {
	ServerRef      string         `json:"serverRef,omitempty"`
	DiscoveryMode  DiscoveryMode  `json:"discoveryMode,omitempty"`
	WatchResources []ResourceKind `json:"watchResources,omitempty"`
	MonitorConfig  MonitorConfig  `json:"monitorConfig,omitempty"`
}

type MonitorInfo struct {
	MonitorID   string `json:"monitorId"`
	URL         string `json:"url"`
	Active      bool   `json:"active"`
	LastUpdated string `json:"lastUpdated"`
}

type KumaMonitorStatus struct {
	Connected      bool          `json:"connected"`
	LastSync       string        `json:"lastSync"`
	Monitors       []MonitorInfo `json:"monitors,omitempty"`
	MonitoredCount int           `json:"monitoredCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kumamonitors,scope=Namespaced,shortName=kumamon
// +kubebuilder:groupName=kumadre.controller
type KumaMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KumaMonitorSpec   `json:"spec,omitempty"`
	Status            KumaMonitorStatus `json:"status,omitempty"`
}

type KumaMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KumaMonitor `json:"items"`
}
