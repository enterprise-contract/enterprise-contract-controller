package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Deployment", func() {
	var (
		dynamicClient dynamic.Interface
		k8sClient     client.Client
		ctx           context.Context
	)

	BeforeEach(func() {
		var err error
		dynamicClient, err = dynamic.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	Describe("CRD Installation", func() {
		It("should have CRDs installed and configured correctly", func() {
			// Verify CRD exists and is accessible
			gvr := schema.GroupVersionResource{
				Group:    "appstudio.redhat.com",
				Version:  "v1alpha1",
				Resource: "enterprisecontractpolicies",
			}
			Eventually(func() error {
				_, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
				return err
			}, time.Second*10, time.Second).Should(Succeed())

			// Verify CRD definition details
			crd := &apiextensionsv1.CustomResourceDefinition{}
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name: "enterprisecontractpolicies.appstudio.redhat.com",
			}, crd)
			Expect(err).NotTo(HaveOccurred())
			Expect(crd.Spec.Group).To(Equal("appstudio.redhat.com"))
			Expect(crd.Spec.Names.Kind).To(Equal("EnterpriseContractPolicy"))
			Expect(crd.Spec.Names.Plural).To(Equal("enterprisecontractpolicies"))
			Expect(crd.Spec.Names.Singular).To(Equal("enterprisecontractpolicy"))
			Expect(crd.Spec.Scope).To(Equal(apiextensionsv1.NamespaceScoped))
			Expect(crd.Spec.Versions).To(HaveLen(1))
			Expect(crd.Spec.Versions[0].Name).To(Equal("v1alpha1"))
			Expect(crd.Spec.Versions[0].Served).To(BeTrue())
			Expect(crd.Spec.Versions[0].Storage).To(BeTrue())
		})
	})
})
