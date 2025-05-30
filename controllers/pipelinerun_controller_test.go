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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("PipelineRun Controller", func() {
	const (
		PipelineRunName      = "test-pipelinerun"
		PipelineRunNamespace = "default"
		timeout              = time.Second * 10
		interval             = time.Millisecond * 250
	)

	var configMap *corev1.ConfigMap

	BeforeEach(func() {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "enterprise-contract-conforma-params",
				Namespace: PipelineRunNamespace,
			},
			Data: map[string]string{
				"GIT_URL":              "https://github.com/enterprise-contract/ec-cli",
				"GIT_REVISION":         "main",
				"GIT_PATH":             "tasks/verify-enterprise-contract/0.1/verify-enterprise-contract.yaml",
				"IGNORE_REKOR":         "true",
				"TIMEOUT":              "60m",
				"WORKERS":              "1",
				"POLICY_CONFIGURATION": "github.com/enterprise-contract/config//slsa3",
				"PUBLIC_KEY":           "k8s://enterprise-contract/public-key",
			},
		}
		Expect(k8sClient.Create(context.Background(), configMap)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), configMap)).Should(Succeed())
	})

	Context("When checking PipelineRun conditions", func() {
		It("Should detect a signed and succeeded PipelineRun", func() {
			By("Creating a signed and succeeded PipelineRun")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName,
					Namespace: PipelineRunNamespace,
					Annotations: map[string]string{
						AnnotationChainsSigned: "true",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			// Set status in-memory (not persisted)
			pipelineRun.Status = tektonv1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: "True",
						},
					},
				},
			}

			By("Checking if the PipelineRun is detected as signed and succeeded")
			Expect(isSignedAndSucceeded(pipelineRun)).Should(BeTrue())
		})

		It("Should not detect an unsigned PipelineRun", func() {
			By("Creating an unsigned but succeeded PipelineRun")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-unsigned",
					Namespace: PipelineRunNamespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   apis.ConditionSucceeded,
								Status: "True",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun is not detected as signed and succeeded")
			Expect(isSignedAndSucceeded(pipelineRun)).Should(BeFalse())
		})

		It("Should not detect a failed PipelineRun", func() {
			By("Creating a signed but failed PipelineRun")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-failed",
					Namespace: PipelineRunNamespace,
					Annotations: map[string]string{
						AnnotationChainsSigned: "true",
					},
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   apis.ConditionSucceeded,
								Status: "False",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun is not detected as signed and succeeded")
			Expect(isSignedAndSucceeded(pipelineRun)).Should(BeFalse())
		})
	})

	Context("When handling VSA completion", func() {
		It("Should detect a PipelineRun with VSA complete", func() {
			By("Creating a PipelineRun with VSA complete annotation")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-vsa-complete",
					Namespace: PipelineRunNamespace,
					Annotations: map[string]string{
						AnnotationVSAComplete: "true",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun is detected as VSA complete")
			Expect(isVSAComplete(pipelineRun)).Should(BeTrue())
		})

		It("Should mark a PipelineRun as VSA complete", func() {
			By("Creating a PipelineRun without VSA complete annotation")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-mark-vsa",
					Namespace: PipelineRunNamespace,
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			By("Marking the PipelineRun as VSA complete")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			Expect(reconciler.markVSAComplete(context.Background(), pipelineRun)).Should(Succeed())

			By("Verifying the PipelineRun has the VSA complete annotation")
			updatedPipelineRun := &tektonv1.PipelineRun{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      PipelineRunName + "-mark-vsa",
				Namespace: PipelineRunNamespace,
			}, updatedPipelineRun)).Should(Succeed())
			Expect(updatedPipelineRun.Annotations[AnnotationVSAComplete]).Should(Equal("true"))
		})
	})

	Context("When reconciling PipelineRuns", func() {
		It("Should process a signed and succeeded PipelineRun", func() {
			By("Creating a signed and succeeded PipelineRun")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-reconcile",
					Namespace: PipelineRunNamespace,
					Annotations: map[string]string{
						AnnotationChainsSigned: "true",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			// Update the status using the status subresource
			pipelineRun.Status = tektonv1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: "True",
						},
					},
				},
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{
							Name:  "IMAGE_URL",
							Value: *tektonv1.NewStructuredValues("quay.io/test/image:latest"),
						},
						{
							Name:  "IMAGE_DIGEST",
							Value: *tektonv1.NewStructuredValues("sha256:1234567890abcdef"),
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(context.Background(), pipelineRun)).Should(Succeed())

			By("Reconciling the PipelineRun")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      PipelineRunName + "-reconcile",
					Namespace: PipelineRunNamespace,
				},
			})
			Expect(err).To(BeNil())

			By("Verifying the PipelineRun has been processed")
			updatedPipelineRun := &tektonv1.PipelineRun{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      PipelineRunName + "-reconcile",
				Namespace: PipelineRunNamespace,
			}, updatedPipelineRun)).Should(Succeed())
			Expect(updatedPipelineRun.Annotations[AnnotationVSAComplete]).Should(Equal("true"))
		})

		It("Should not process an already processed PipelineRun", func() {
			By("Creating a PipelineRun that's already been processed")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-already-processed",
					Namespace: PipelineRunNamespace,
					Annotations: map[string]string{
						AnnotationChainsSigned: "true",
						AnnotationVSAComplete:  "true",
					},
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   apis.ConditionSucceeded,
								Status: "True",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			By("Reconciling the PipelineRun")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      PipelineRunName + "-already-processed",
					Namespace: PipelineRunNamespace,
				},
			})
			Expect(err).Should(BeNil())

			By("Verifying the PipelineRun hasn't been processed again")
			updatedPipelineRun := &tektonv1.PipelineRun{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      PipelineRunName + "-already-processed",
				Namespace: PipelineRunNamespace,
			}, updatedPipelineRun)).Should(Succeed())
			Expect(updatedPipelineRun.Annotations[AnnotationVSAComplete]).Should(Equal("true"))
		})
	})

	Context("When triggering Conforma verification", func() {
		It("Should create a TaskRun with correct parameters", func() {
			By("Creating a PipelineRun with required results")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-conforma",
					Namespace: PipelineRunNamespace,
					Annotations: map[string]string{
						"conforma/source-data": "test-source-data",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			// Update the status with results
			pipelineRun.Status = tektonv1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{
							Name:  "IMAGE_URL",
							Value: *tektonv1.NewStructuredValues("quay.io/test/image:latest"),
						},
						{
							Name:  "IMAGE_DIGEST",
							Value: *tektonv1.NewStructuredValues("sha256:1234567890abcdef"),
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(context.Background(), pipelineRun)).Should(Succeed())

			By("Triggering Conforma verification")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			Expect(reconciler.triggerConforma(context.Background(), pipelineRun)).Should(Succeed())

			By("Verifying the TaskRun was created with correct parameters")
			taskRunList := &tektonv1.TaskRunList{}
			Expect(k8sClient.List(context.Background(), taskRunList, client.InNamespace(PipelineRunNamespace))).Should(Succeed())

			var foundTaskRun *tektonv1.TaskRun
			for i := range taskRunList.Items {
				tr := &taskRunList.Items[i]
				if tr.Labels["enterprise-contract.redhat.com/pipelinerun"] == pipelineRun.Name {
					foundTaskRun = tr
					break
				}
			}
			Expect(foundTaskRun).ToNot(BeNil(), "Expected to find a TaskRun for the PipelineRun")
			taskRun := *foundTaskRun
			Expect(taskRun.Labels["app.kubernetes.io/created-by"]).To(Equal("enterprise-contract-controller"))
			Expect(taskRun.Labels["enterprise-contract.redhat.com/pipelinerun"]).To(Equal(pipelineRun.Name))
			Expect(string(taskRun.Spec.TaskRef.Resolver)).To(Equal("git"))
			Expect(taskRun.Spec.TaskRef.ResolverRef.Params).To(ContainElements(
				tektonv1.Param{
					Name: "url",
					Value: tektonv1.ParamValue{
						StringVal: "https://github.com/enterprise-contract/ec-cli",
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "revision",
					Value: tektonv1.ParamValue{
						StringVal: "main",
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "pathInRepo",
					Value: tektonv1.ParamValue{
						StringVal: "tasks/verify-enterprise-contract/0.1/verify-enterprise-contract.yaml",
						Type:      tektonv1.ParamTypeString,
					},
				},
			))
			Expect(taskRun.Spec.Params).To(ContainElements(
				tektonv1.Param{
					Name: "IMAGES",
					Value: tektonv1.ParamValue{
						StringVal: `{"components":[{"name": "quay.io/test/image:latest", "containerImage":"quay.io/test/image:latest@sha256:1234567890abcdef"}]}`,
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "IGNORE_REKOR",
					Value: tektonv1.ParamValue{
						StringVal: "true",
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "TIMEOUT",
					Value: tektonv1.ParamValue{
						StringVal: "60m",
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "WORKERS",
					Value: tektonv1.ParamValue{
						StringVal: "1",
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "POLICY_CONFIGURATION",
					Value: tektonv1.ParamValue{
						StringVal: "github.com/enterprise-contract/config//slsa3",
						Type:      tektonv1.ParamTypeString,
					},
				},
				tektonv1.Param{
					Name: "PUBLIC_KEY",
					Value: tektonv1.ParamValue{
						StringVal: "k8s://enterprise-contract/public-key",
						Type:      tektonv1.ParamTypeString,
					},
				},
			))
			Expect(taskRun.Spec.Timeout.Duration).To(Equal(10 * time.Minute))
		})

		It("Should create TaskRun with correct parameters when PipelineRun has no annotations", func() {
			By("Creating a PipelineRun without annotations")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-conforma-default",
					Namespace: PipelineRunNamespace,
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			// Update the status with results
			pipelineRun.Status = tektonv1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{
							Name:  "IMAGE_URL",
							Value: *tektonv1.NewStructuredValues("quay.io/test/image:latest"),
						},
						{
							Name:  "IMAGE_DIGEST",
							Value: *tektonv1.NewStructuredValues("sha256:1234567890abcdef"),
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(context.Background(), pipelineRun)).Should(Succeed())

			By("Triggering Conforma verification")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			Expect(reconciler.triggerConforma(context.Background(), pipelineRun)).Should(Succeed())

			By("Verifying the TaskRun was created with correct parameters")
			taskRunList := &tektonv1.TaskRunList{}
			Expect(k8sClient.List(context.Background(), taskRunList, client.InNamespace(PipelineRunNamespace))).Should(Succeed())

			var foundTaskRun *tektonv1.TaskRun
			for i := range taskRunList.Items {
				tr := &taskRunList.Items[i]
				if tr.Labels["enterprise-contract.redhat.com/pipelinerun"] == pipelineRun.Name {
					foundTaskRun = tr
					break
				}
			}
			Expect(foundTaskRun).ToNot(BeNil(), "Expected to find a TaskRun for the PipelineRun")
			taskRun := *foundTaskRun
			Expect(taskRun.Spec.Params).To(ContainElement(
				tektonv1.Param{
					Name: "IMAGES",
					Value: tektonv1.ParamValue{
						StringVal: `{"components":[{"name": "quay.io/test/image:latest", "containerImage":"quay.io/test/image:latest@sha256:1234567890abcdef"}]}`,
						Type:      tektonv1.ParamTypeString,
					},
				},
			))
		})

		It("Should fail when IMAGE_URL result is missing", func() {
			By("Creating a PipelineRun without IMAGE_URL result")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-conforma-no-url",
					Namespace: PipelineRunNamespace,
				},
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						Results: []tektonv1.PipelineRunResult{
							{
								Name:  "IMAGE_DIGEST",
								Value: *tektonv1.NewStructuredValues("sha256:1234567890abcdef"),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			By("Triggering Conforma verification")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			err := reconciler.triggerConforma(context.Background(), pipelineRun)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("IMAGE_URL result not found"))
		})

		It("Should fail when IMAGE_DIGEST result is missing", func() {
			By("Creating a PipelineRun without IMAGE_DIGEST result")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-conforma-no-digest",
					Namespace: PipelineRunNamespace,
				},
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						Results: []tektonv1.PipelineRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *tektonv1.NewStructuredValues("quay.io/test/image:latest"),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), pipelineRun)).Should(Succeed())

			// Update the status with only IMAGE_URL result
			pipelineRun.Status = tektonv1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{
							Name:  "IMAGE_URL",
							Value: *tektonv1.NewStructuredValues("quay.io/test/image:latest"),
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(context.Background(), pipelineRun)).Should(Succeed())

			By("Triggering Conforma verification")
			reconciler := &PipelineRunReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			err := reconciler.triggerConforma(context.Background(), pipelineRun)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("IMAGE_DIGEST result not found in PipelineRun %s", pipelineRun.Name)))
		})
	})
})
