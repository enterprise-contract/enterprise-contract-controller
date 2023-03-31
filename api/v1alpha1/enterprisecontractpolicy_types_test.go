/*
Copyright 2023.

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
	"strings"
	"testing"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestArbitraryRuleDataEncode(t *testing.T) {
	ecp := EnterpriseContractPolicy{
		Spec: EnterpriseContractPolicySpec{
			Sources: []Source{
				{
					RuleData: &v1.JSON{Raw: []byte(`{"my":"data","here":14}`)},
				},
			},
		},
	}

	out := strings.Builder{}
	err := unstructured.UnstructuredJSONScheme.Encode(&ecp, &out)
	if err != nil {
		t.Fatalf("Unexpected error when encoding: %s", err)
	}

	expected := `{"metadata":{"creationTimestamp":null},"spec":{"sources":[{"ruleData":{"my":"data","here":14}}]},"status":{}}` + "\n"
	got := out.String()
	if got != expected {
		t.Errorf("Expecting encoded to be: %s, but it was %s", expected, got)
	}
}

func TestArbitraryRuleDataDecode(t *testing.T) {
	ecp := EnterpriseContractPolicy{}
	_, _, err := unstructured.UnstructuredJSONScheme.Decode([]byte(`{"apiVersion":"appstudio.redhat.com/v1alpha1","kind":"EnterpriseContractPolicy","spec":{"sources":[{"ruleData":{"my":"data","here":14}}]}}`), nil, &ecp)
	if err != nil {
		t.Fatalf("Unexpected error when encoding: %s", err)
	}

	expected := `{"my":"data","here":14}`
	got := string(ecp.Spec.Sources[0].RuleData.Raw)
	if got != expected {
		t.Errorf("Expecting decoded to be: %s, but it was %s", expected, got)
	}
}
