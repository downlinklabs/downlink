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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyntheticTestSource struct {
	Inline string `json:"inline,omitempty"`
}

type ArtifactStorageSpec struct {
	// Enabled is the feature gate. When false all other fields are ignored.
	Enabled bool `json:"enabled"`

	// Bucket is the object storage bucket name. Required when Enabled is true.
	// +kubebuilder:validation:Optional
	Bucket string `json:"bucket,omitempty"`

	// Endpoint is the S3-compatible API endpoint URL.
	// Leave empty to use the AWS S3 default endpoint.
	// Set for MinIO (e.g. "http://minio.minio-ns.svc:9000") or GCS S3 interop
	// (e.g. "https://storage.googleapis.com").
	// +kubebuilder:validation:Optional
	Endpoint string `json:"endpoint,omitempty"`

	// Region is the storage region (e.g. "us-east-2"). Optional for MinIO.
	// +kubebuilder:validation:Optional
	Region string `json:"region,omitempty"`

	// CredentialsSecret names a Secret in the same namespace containing keys
	// "accessKey" and "secretKey" for static credential authentication.
	// The operator does not create this Secret; it must already exist.
	// +kubebuilder:validation:Optional
	CredentialsSecret string `json:"credentialsSecret,omitempty"`

	// UseSDKCreds instructs the artifact store to use the AWS SDK default
	// credential chain (environment variables, EC2 instance profile, ECS task
	// role, etc.) instead of explicit credentials. Set this to true when running
	// on EC2 with an instance profile or in any environment where ambient
	// credentials are available. CredentialsSecret is ignored when true.
	// +kubebuilder:validation:Optional
	UseSDKCreds bool `json:"useSDKCreds,omitempty"`

	// ServiceAccountAnnotations are merged onto the workflow ServiceAccount.
	// Use {"eks.amazonaws.com/role-arn": "<arn>"} for AWS IRSA (EKS only).
	// Use {"iam.gke.io/gcp-service-account": "<email>"} for GKE Workload Identity.
	// When non-empty, also implies UseSDKCreds. CredentialsSecret is ignored.
	// +kubebuilder:validation:Optional
	ServiceAccountAnnotations map[string]string `json:"serviceAccountAnnotations,omitempty"`
}

type SyntheticTestSpec struct {
	Schedule           string               `json:"schedule,omitempty"`
	Source             SyntheticTestSource  `json:"source,omitempty"`
	ServiceAccountName *string              `json:"serviceAccountName,omitempty"`
	Entrypoint         string               `json:"entrypoint,omitempty"`
	Container          v1.Container         `json:"container,omitempty"`
	ArtifactStorage    *ArtifactStorageSpec `json:"artifactStorage,omitempty"`
}

// SyntheticTestStatus defines the observed state of SyntheticTest.
type SyntheticTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SyntheticTest is the Schema for the synthetictests API.
type SyntheticTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SyntheticTestSpec   `json:"spec,omitempty"`
	Status SyntheticTestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SyntheticTestList contains a list of SyntheticTest.
type SyntheticTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SyntheticTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SyntheticTest{}, &SyntheticTestList{})
}
