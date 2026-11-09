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

type SyntheticTestRunSpec struct {
	Name         string       `json:"name"`
	WorkflowName string       `json:"workflowName"`
	StartedAt    *metav1.Time `json:"startedAt,omitempty"`
}

type SyntheticTestRunStatus struct {
	Phase       string       `json:"phase"` // Succeeded, Failed, Running
	FinishedAt  *metav1.Time `json:"finishedAt,omitempty"`
	Message     string       `json:"message,omitempty"`
	ArtifactURL string       `json:"artifactURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SyntheticTestRun is the Schema for the synthetictestruns API.
type SyntheticTestRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SyntheticTestRunSpec   `json:"spec,omitempty"`
	Status SyntheticTestRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SyntheticTestRunList contains a list of SyntheticTestRun.
type SyntheticTestRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SyntheticTestRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SyntheticTestRun{}, &SyntheticTestRunList{})
}
