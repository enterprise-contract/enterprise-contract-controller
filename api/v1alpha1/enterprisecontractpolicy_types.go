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

// EnterpriseContractPolicySpec is used to configure the Enterprise Contract Policy
type EnterpriseContractPolicySpec struct {
	// Description of the policy or it's intended use
	// +optional
	Description string `json:"description,omitempty"`
	// Sources of the policy
	// +kubebuilder:validation:MinItems:=1
	Sources []PolicySource `json:"sources"`
	// Authorization for per component release approvals
	// +optional
	Authorization *Authorization `json:"authorization,omitempty"`
	// Exceptions under which the policy is evaluated as successful even if the listed policy checks have reported failure
	// +optional
	Exceptions *EnterpriseContractPolicyExceptions `json:"exceptions,omitempty"`
	// URL of the Rekor instance. Empty string disables Rekor integration
	// +optional
	RekorUrl string `json:"rekorUrl,omitempty"`
	// Public key used to validate the signature of images and attestations
	// +optional
	PublicKey string `json:"publicKey,omitempty"`
}

// Authorization defines a release approval
type Authorization struct {
	// Components based authorization
	// +optional
	Components []AuthorizedComponent `json:"components,omitempty"`
}

// Authorization defines a release approval on a component basis
type AuthorizedComponent struct {
	// ChangeID is the identifier of the change, e.g. git commit id
	// +optional
	ChangeID string `json:"changeId,omitempty"`
	// Repository of the component sources
	// +optional
	Repository string `json:"repository,omitempty"`
	// Authorizer is the email address of the person authorizing the release
	// +optional
	Authorizer string `json:"authorizer,omitempty"`
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
	Revision string `json:"revision,omitempty"`
}

// EnterpriseContractPolicyExceptions configuration of exceptions for the policy evaluation
type EnterpriseContractPolicyExceptions struct {
	// NonBlocking set of policy exceptions that in case of failure do not block
	// the success of the outcome
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
