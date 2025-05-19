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
	"time"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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

func (r *PipelineRunReconciler) triggerConforma(ctx context.Context, pr *tektonv1.PipelineRun) error {
	log := log.FromContext(ctx)

	// These values should come from the PipelineRun or a config map / policy
	sourceData := pr.Annotations["conforma/source-data"]
	if sourceData == "" {
		sourceData = "default-source-data"
	}
	snapshotFile := pr.Annotations["conforma/snapshot-filename"]
	if snapshotFile == "" {
		snapshotFile = "default-snapshot.json"
	}

	taskRun := &tektonv1.TaskRun{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "conforma-verify-",
			Namespace:    pr.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/created-by":               "enterprise-contract-controller",
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
						{Name: "pathInRepo", Value: tektonv1.ParamValue{StringVal: "tasks/verify-conforma-konflux-ta.yaml", Type: tektonv1.ParamTypeString}},
					},
				},
			},
			Params: []tektonv1.Param{
				{Name: "SOURCE_DATA_ARTIFACT", Value: *tektonv1.NewStructuredValues(sourceData)},
				{Name: "SNAPSHOT_FILENAME", Value: *tektonv1.NewStructuredValues(snapshotFile)},
				// Add additional params as needed, possibly pulled from PipelineRun annotations or ConfigMap
				{Name: "POLICY_CONFIGURATION", Value: *tektonv1.NewStructuredValues("enterprise-contract-service/default")},
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
