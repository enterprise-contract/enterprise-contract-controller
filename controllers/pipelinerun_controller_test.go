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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("PipelineRun Controller", func() {
	const (
		PipelineRunName      = "test-pipelinerun"
		PipelineRunNamespace = "default"
		timeout              = time.Second * 10
		interval             = time.Millisecond * 250
	)

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
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						Results: []tektonv1.PipelineRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *tektonv1.NewStructuredValues("quay.io/test/image"),
							},
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
			Expect(reconciler.triggerConforma(context.Background(), pipelineRun)).Should(Succeed())

			By("Verifying the TaskRun was created with correct parameters")
			taskRunList := &tektonv1.TaskRunList{}
			Expect(k8sClient.List(context.Background(), taskRunList, client.InNamespace(PipelineRunNamespace))).Should(Succeed())
			Expect(taskRunList.Items).To(HaveLen(1))

			taskRun := taskRunList.Items[0]
			Expect(taskRun.Labels).To(HaveKeyWithValue("app.kubernetes.io/created-by", "enterprise-contract-controller"))
			Expect(taskRun.Labels).To(HaveKeyWithValue("enterprise-contract.redhat.com/pipelinerun", PipelineRunName+"-conforma"))

			// Verify TaskRun parameters
			Expect(taskRun.Spec.Params).To(ContainElement(HaveField("Name", "SOURCE_DATA_ARTIFACT")))
			Expect(taskRun.Spec.Params).To(ContainElement(HaveField("Name", "SNAPSHOT_FILENAME")))
			Expect(taskRun.Spec.Params).To(ContainElement(HaveField("Name", "POLICY_CONFIGURATION")))

			// Verify TaskRef
			Expect(taskRun.Spec.TaskRef.ResolverRef.Resolver).To(Equal("git"))
			Expect(taskRun.Spec.TaskRef.ResolverRef.Params).To(ContainElement(HaveField("Name", "url")))
			Expect(taskRun.Spec.TaskRef.ResolverRef.Params).To(ContainElement(HaveField("Name", "revision")))
			Expect(taskRun.Spec.TaskRef.ResolverRef.Params).To(ContainElement(HaveField("Name", "pathInRepo")))
		})

		It("Should fail when IMAGE_URL is missing", func() {
			By("Creating a PipelineRun without IMAGE_URL result")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-missing-url",
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

		It("Should fail when IMAGE_DIGEST is missing", func() {
			By("Creating a PipelineRun without IMAGE_DIGEST result")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-missing-digest",
					Namespace: PipelineRunNamespace,
				},
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						Results: []tektonv1.PipelineRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *tektonv1.NewStructuredValues("quay.io/test/image"),
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
			Expect(err.Error()).To(ContainSubstring("IMAGE_DIGEST result not found"))
		})

		It("Should use default source data when not specified", func() {
			By("Creating a PipelineRun without source data annotation")
			pipelineRun := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineRunName + "-default-source",
					Namespace: PipelineRunNamespace,
				},
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						Results: []tektonv1.PipelineRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *tektonv1.NewStructuredValues("quay.io/test/image"),
							},
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
			Expect(reconciler.triggerConforma(context.Background(), pipelineRun)).Should(Succeed())

			By("Verifying the TaskRun was created with default source data")
			taskRunList := &tektonv1.TaskRunList{}
			Expect(k8sClient.List(context.Background(), taskRunList, client.InNamespace(PipelineRunNamespace))).Should(Succeed())
			Expect(taskRunList.Items).To(HaveLen(1))

			taskRun := taskRunList.Items[0]
			Expect(taskRun.Spec.Params).To(ContainElement(HaveField("Value.StringVal", "default-source-data")))
		})
	})
})
