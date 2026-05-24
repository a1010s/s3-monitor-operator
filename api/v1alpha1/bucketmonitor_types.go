/*
Copyright 2026.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BucketMonitorSpec defines the desired state of BucketMonitor
type BucketMonitorSpec struct {
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// +kubebuilder:validation:Required
	CredentialsSecret string `json:"credentialsSecret"`

	// +kubebuilder:default="5m"
	// +optional
	Interval string `json:"interval,omitempty"`

	// +optional
	AlertThresholdMB *int64 `json:"alertThresholdMB,omitempty"`
}

// BucketMonitorStatus defines the observed state of BucketMonitor.
type BucketMonitorStatus struct {
	TotalSizeBytes int64       `json:"totalSizeBytes,omitempty"`
	ObjectCount    int64       `json:"objectCount,omitempty"`
	TotalSizeHuman string      `json:"totalSizeHuman,omitempty"`
	LastChecked    metav1.Time `json:"lastChecked,omitempty"`
	Phase          string      `json:"phase,omitempty"`
	Message        string      `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Bucket",type="string",JSONPath=".spec.bucket"
// +kubebuilder:printcolumn:name="Size",type="string",JSONPath=".status.totalSizeHuman"
// +kubebuilder:printcolumn:name="Objects",type="integer",JSONPath=".status.objectCount"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="LastChecked",type="date",JSONPath=".status.lastChecked"

// BucketMonitor is the Schema for the bucketmonitors API
type BucketMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketMonitorSpec   `json:"spec,omitempty"`
	Status BucketMonitorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BucketMonitorList contains a list of BucketMonitor
type BucketMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BucketMonitor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BucketMonitor{}, &BucketMonitorList{})
}
