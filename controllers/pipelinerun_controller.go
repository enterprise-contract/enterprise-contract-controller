/*
Copyright 2024.

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

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// AnnotationChainsSigned is the annotation that indicates a PipelineRun has been signed by Chains
	AnnotationChainsSigned = "chains.tekton.dev/signed"
	// AnnotationVSAComplete is the annotation that indicates VSA generation is complete
	AnnotationVSAComplete = "enterprise-contract.redhat.com/vsa-complete"
	// ConfigMapName is the name of the ConfigMap containing Conforma parameters
	ConfigMapName = "enterprise-contract-conforma-params"
)

// PipelineRunReconciler reconciles PipelineRun objects
type PipelineRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/status,verbs=get

// Reconcile handles the reconciliation loop for PipelineRuns
func (r *PipelineRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling PipelineRun", "name", req.NamespacedName)

	// Fetch the PipelineRun
	pipelineRun := &tektonv1.PipelineRun{}
	if err := r.Get(ctx, req.NamespacedName, pipelineRun); err != nil {
		log.Error(err, "Failed to fetch PipelineRun")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the PipelineRun is signed and has succeeded
	if !isSignedAndSucceeded(pipelineRun) {
		log.Info("PipelineRun is not signed and succeeded, skipping", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Check if VSA generation is already complete
	if isVSAComplete(pipelineRun) {
		log.Info("PipelineRun is already processed, skipping", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Trigger Conforma (stubbed for now)
	if err := r.triggerConforma(ctx, pipelineRun); err != nil {
		log.Error(err, "Failed to trigger Conforma")
		return ctrl.Result{}, err
	}

	// Mark VSA generation as complete
	if err := r.markVSAComplete(ctx, pipelineRun); err != nil {
		log.Error(err, "Failed to mark VSA generation as complete")
		return ctrl.Result{}, err
	}

	log.Info("PipelineRun processed successfully", "name", req.NamespacedName)
	return ctrl.Result{}, nil
}

// isSignedAndSucceeded checks if the PipelineRun is signed and has succeeded
func isSignedAndSucceeded(pr *tektonv1.PipelineRun) bool {
	log := log.FromContext(context.Background())

	// Check if the PipelineRun has succeeded
	condition := pr.Status.GetCondition(apis.ConditionSucceeded)
	if condition == nil {
		log.Info("PipelineRun has no succeeded condition")
		return false
	}
	if condition.Status != "True" {
		log.Info("PipelineRun succeeded condition is not True", "status", condition.Status)
		return false
	}

	// Check if the PipelineRun is signed
	signed, ok := pr.Annotations[AnnotationChainsSigned]
	if !ok {
		log.Info("PipelineRun has no signed annotation")
		return false
	}
	if signed != "true" {
		log.Info("PipelineRun signed annotation is not true", "value", signed)
		return false
	}

	log.Info("PipelineRun is signed and succeeded")
	return true
}

// isVSAComplete checks if VSA generation is already complete
func isVSAComplete(pr *tektonv1.PipelineRun) bool {
	if complete, ok := pr.Annotations[AnnotationVSAComplete]; ok && complete == "true" {
		return true
	}
	return false
}

// getControllerNamespace returns the namespace where the controller is running
func getControllerNamespace() string {
	// Read namespace from the downward API file
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		// Fallback to default if we can't read the file
		return "default"
	}
	return string(nsBytes)
}

func (r *PipelineRunReconciler) triggerConforma(ctx context.Context, pr *tektonv1.PipelineRun) error {
	log := log.FromContext(ctx)

	// Get IMAGE_URL and IMAGE_DIGEST from PipelineRun results
	imageURL := ""
	imageDigest := ""
	for _, result := range pr.Status.PipelineRunStatusFields.Results {
		switch result.Name {
		case "IMAGE_URL":
			imageURL = result.Value.StringVal
		case "IMAGE_DIGEST":
			imageDigest = result.Value.StringVal
		}
	}

	if imageURL == "" {
		return fmt.Errorf("IMAGE_URL result not found in PipelineRun %s", pr.Name)
	}
	if imageDigest == "" {
		return fmt.Errorf("IMAGE_DIGEST result not found in PipelineRun %s", pr.Name)
	}

	// Create JSON string for SNAPSHOT_FILENAME
	snapshotJSON := fmt.Sprintf(`{"components":[{"name": "%s", "containerImage":"%s@%s"}]}`, imageURL, imageURL, imageDigest)

	// get param values from a configmap
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Name: ConfigMapName, Namespace: getControllerNamespace()}, configMap); err != nil {
		log.Error(err, "Failed to get Conforma params configmap")
		return err
	}

	taskRun := &tektonv1.TaskRun{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "conforma-verify-",
			Namespace:    getControllerNamespace(),
			Labels: map[string]string{
				"app.kubernetes.io/created-by": "enterprise-contract-controller",
				"enterprise-contract.redhat.com/pipelinerun": pr.Name,
			},
		},
		Spec: tektonv1.TaskRunSpec{
			TaskRef: &tektonv1.TaskRef{
				ResolverRef: tektonv1.ResolverRef{
					Resolver: "git",
					Params: []tektonv1.Param{
						{Name: "url", Value: tektonv1.ParamValue{StringVal: "https://github.com/enterprise-contract/ec-cli", Type: tektonv1.ParamTypeString}},
						{Name: "revision", Value: tektonv1.ParamValue{StringVal: "main", Type: tektonv1.ParamTypeString}},
						{Name: "pathInRepo", Value: tektonv1.ParamValue{StringVal: "tasks/verify-enterprise-contract/0.1/verify-enterprise-contract.yaml", Type: tektonv1.ParamTypeString}},
					},
				},
			},

			Params: []tektonv1.Param{
				{Name: "IMAGES", Value: *tektonv1.NewStructuredValues(snapshotJSON)},
				// apply these values from the configmap
				{Name: "IGNORE_REKOR", Value: *tektonv1.NewStructuredValues(configMap.Data["IGNORE_REKOR"])},
				{Name: "TIMEOUT", Value: *tektonv1.NewStructuredValues(configMap.Data["TIMEOUT"])},
				{Name: "WORKERS", Value: *tektonv1.NewStructuredValues(configMap.Data["WORKERS"])},
				{Name: "POLICY_CONFIGURATION", Value: *tektonv1.NewStructuredValues(configMap.Data["POLICY_CONFIGURATION"])},
				{Name: "PUBLIC_KEY", Value: *tektonv1.NewStructuredValues(configMap.Data["PUBLIC_KEY"])},
			},
			Timeout: &v1.Duration{Duration: 10 * time.Minute},
		},
	}

	if err := r.Client.Create(ctx, taskRun); err != nil {
		log.Error(err, "Failed to create Conforma TaskRun")
		return err
	}

	log.Info("Conforma TaskRun created", "name", taskRun.Name)
	return nil
}

// markVSAComplete marks the PipelineRun as having completed VSA generation
func (r *PipelineRunReconciler) markVSAComplete(ctx context.Context, pr *tektonv1.PipelineRun) error {
	patch := client.MergeFrom(pr.DeepCopy())
	if pr.Annotations == nil {
		pr.Annotations = make(map[string]string)
	}
	pr.Annotations[AnnotationVSAComplete] = "true"
	return r.Patch(ctx, pr, patch)
}

// SetupWithManager sets up the controller with the Manager
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tektonv1.PipelineRun{}).
		Complete(r)
}
