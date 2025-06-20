package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/controller"
)

var (
	orgNamespace = "default"
)

var _ = Describe("Org Controller", func() {
	Context("When reconciling an org resource", func() {
		const orgName = "test-org-1"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      orgName,
			Namespace: orgNamespace,
		}
		org := &platformv1alpha1.Org{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Org")
			err := k8sClient.Get(ctx, typeNamespacedName, org)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Org{
					ObjectMeta: metav1.ObjectMeta{
						Name:      orgName,
						Namespace: orgNamespace,
					},
					Spec: platformv1alpha1.OrgSpec{
						OrgQuota: platformv1alpha1.OrgQuota{
							Clusters: 10,
							Apps:     10,
						},
						DisplayName: orgName + "-display-name",
						OwnerEmail:  "test@test.com",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &platformv1alpha1.Org{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Org")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource if crd is created", func() {
			By("Reconciling the created resource")
			controllerReconciler := buildTestReconciler()

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				var updated platformv1alpha1.Org
				err := k8sClient.Get(ctx, typeNamespacedName, &updated)
				g.Expect(err).ToNot(HaveOccurred())

				cond := apimeta.FindStatusCondition(updated.Status.Conditions, platformv1alpha1.Ready)
				g.Expect(cond).NotTo(BeNil())
				g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(cond.Reason).To(Equal("Reconciled"))
				g.Expect(cond.Message).To(Equal("Org is ready"))

			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
		})

	})
})

// helper function to build a test reconciler
func buildTestReconciler() *controller.OrgReconciler {
	// A reconciler needs caches + metrics – we provide them explicitly
	return &controller.OrgReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}
