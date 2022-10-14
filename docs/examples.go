package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"

	ecc "github.com/hacbs-contract/enterprise-contract-controller/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func simplePolicy() *ecc.EnterpriseContractPolicySpec {
	return &ecc.EnterpriseContractPolicySpec{
		Description: "ACME & co policy",
		Sources: []string{
			"git::https://github.com/acme/ec-policy.git//policy?ref=prod",
		},
		Exceptions: &ecc.EnterpriseContractPolicyExceptions{
			NonBlocking: []string{
				"friday_policy",
				"room_temperature",
			},
		},
	}
}

func generateJSONSpecExample() (err error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err = encoder.Encode(simplePolicy()); err == nil {
		err = os.WriteFile(path.Join("docs", "modules", "ROOT", "examples", "spec-example.json"), out.Bytes(), 0644)
	}

	return
}

func generateK8SYAMLExample() (err error) {
	policy := ecc.EnterpriseContractPolicy{
		TypeMeta: v1.TypeMeta{
			Kind:       "EnterpriseContractPolicy",
			APIVersion: ecc.GroupVersion.Identifier(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "ec-policy",
			Namespace: "acme",
		},
		Spec: *simplePolicy(),
	}

	var out []byte
	if out, err = yaml.Marshal(policy); err == nil {
		err = os.WriteFile(path.Join("docs", "modules", "ROOT", "examples", "k8s-example.yaml"), out, 0644)
	}

	return
}

func main() {
	generators := []func() error{
		generateJSONSpecExample,
		generateK8SYAMLExample,
	}

	for _, g := range generators {
		if err := g(); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to generate example: %v", err)
			os.Exit(1)
		}
	}
}
