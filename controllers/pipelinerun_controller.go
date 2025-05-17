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

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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

// triggerConforma triggers the Conforma validation (stubbed for now)
func (r *PipelineRunReconciler) triggerConforma(ctx context.Context, pr *tektonv1.PipelineRun) error {
	// TODO: Implement actual Conforma trigger logic
	return nil
}

// markVSAComplete marks the PipelineRun as having completed VSA generation
func (r *PipelineRunReconciler) markVSAComplete(ctx context.Context, pr *tektonv1.PipelineRun) error {
	if pr.Annotations == nil {
		pr.Annotations = make(map[string]string)
	}
	pr.Annotations[AnnotationVSAComplete] = "true"
	return r.Update(ctx, pr)
}

// SetupWithManager sets up the controller with the Manager
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tektonv1.PipelineRun{}).
		Complete(r)
}
