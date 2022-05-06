/*
Copyright 2022.

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

// Important: Run "make" to regenerate code after modifying this file
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EnterpriseContractPolicySpec represents the desired state of EnterpriseContractPolicy
type EnterpriseContractPolicySpec struct {
	// Description text describing the the policy or it's intended use
	// +optional
	Description *string `json:"description"`
	// Sources is list of policy sources
	// +kubebuilder:validation:MinItems:=1
	Sources []PolicySource `json:"sources"`
	// Exceptions configures exceptions under which the policy is evaluated as successful even if the listed policy checks have reported failure
	// +optional
	Exceptions *EnterpriseContractPolicyExceptions `json:"exceptions,omitempty"`
}

// PolicySource represents the configuration of the source for the policy
type PolicySource struct {
	// GitRepository configures fetching of the policies from a Git repository
	// +optional
	GitRepository *GitPolicySource `json:"git,omitempty"`
}

type GitPolicySource struct {
	// Repository URL
	Repository string `json:"repository"`
	// Revision matching the branch, commit id or similar to fetch. Defaults to `main`
	// +kubebuilder:default:=main
	// +optional
	Revision *string `json:"revision"`
}

// EnterpriseContractPolicyExceptions configuration of exceptions for the policy evaluation
type EnterpriseContractPolicyExceptions struct {
	// +optional
	// +listType:=set
	NonBlocking []string `json:"nonBlocking,omitempty"`
}

// EnterpriseContractPolicyStatus defines the observed state of EnterpriseContractPolicy
type EnterpriseContractPolicyStatus struct {
	// TODO what to add here?
	// ideas;
	// - on what the policy was applied
	// - history of changes
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=all
// +kubebuilder:resource:shortName=ecp
// +kubebuilder:subresource:status
// EnterpriseContractPolicy is the Schema for the enterprisecontractpolicies API
type EnterpriseContractPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnterpriseContractPolicySpec   `json:"spec,omitempty"`
	Status EnterpriseContractPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnterpriseContractPolicyList contains a list of EnterpriseContractPolicy
type EnterpriseContractPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnterpriseContractPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnterpriseContractPolicy{}, &EnterpriseContractPolicyList{})
}
