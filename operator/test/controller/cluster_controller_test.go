package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	controllerImpl "github.com/mofe64/vulkan/operator/internal/controller"
	"github.com/mofe64/vulkan/operator/internal/metrics"
	utils "github.com/mofe64/vulkan/operator/internal/utils"
	testUtils "github.com/mofe64/vulkan/operator/test/utils"
)

var (
	clusterNamespace    = "default"
	clusterSecretName   = "test-kubeconfig"
	clusterNodeName     = "fake-node-0"
	orgName             = "test-org"
	clusterType         = "attached"
	targetClientFactory utils.TargetClientFactory
)

//Note -> orgs and clusters are created in the same namespace

// helper: stub TargetClientFactory so we create error scenarios
type noNodeTargetClusterClientFactory struct{}

func (f *noNodeTargetClusterClientFactory) ClientFor(ctx context.Context, _ *platformv1alpha1.Cluster) (client.Client, error) {
	// client-with-empty-cache: return an in-memory client with no Node objects
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).Build(), nil
}

var _ = Describe("Cluster Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: clusterNamespace,
		}
		cluster := &platformv1alpha1.Cluster{}

		BeforeEach(func() {
			By("creating the cluster resource and it's kubeconfig secret for the test Cluster")

			// create the  org resource
			org := &platformv1alpha1.Org{
				ObjectMeta: metav1.ObjectMeta{Name: orgName, Namespace: clusterNamespace},
				Spec: platformv1alpha1.OrgSpec{
					OrgQuota:    platformv1alpha1.OrgQuota{Clusters: 10, Apps: 10},
					DisplayName: orgName + "-display-name",
					OwnerEmail:  "test@test.com",
				},
			}
			Expect(k8sClient.Create(ctx, org)).To(Succeed())

			// create a kubeconfig for the test cluster
			kcBytes, err := testUtils.KubeconfigWithEmbeddedCA(testEnv.Config)
			Expect(err).NotTo(HaveOccurred())
			// create a kubeconfig secret for the test cluster
			kcfg := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: clusterNamespace,
				},
				Data: map[string][]byte{"kubeconfig": kcBytes},
			}
			Expect(k8sClient.Create(ctx, kcfg)).To(Succeed())

			// create a fake ready node for the test cluster
			ready := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: clusterNodeName},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{{
						Type:               corev1.NodeReady,
						Status:             corev1.ConditionTrue,
						LastHeartbeatTime:  metav1.Now(),
						LastTransitionTime: metav1.Now(),
					}},
				},
			}
			Expect(k8sClient.Create(ctx, ready)).To(Succeed())

			// create the cluster resource
			err = k8sClient.Get(ctx, typeNamespacedName, cluster)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: clusterNamespace,
					},
					Spec: platformv1alpha1.ClusterSpec{
						OrgRef:           orgName,
						Type:             clusterType,
						KubeconfigSecret: clusterSecretName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			// create the target client factory
			targetClientFactory = utils.NewTargetClientFactory(k8sClient)
		})

		AfterEach(func() {

			resource := &platformv1alpha1.Cluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the test Cluster")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			// delete the kubeconfig secret
			Expect(k8sClient.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: clusterNamespace,
				},
			})).To(Succeed())

			// delete the fake node
			Expect(k8sClient.Delete(ctx, &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: clusterNodeName},
			})).To(Succeed())

			// delete the org resource
			Expect(k8sClient.Delete(ctx, &platformv1alpha1.Org{
				ObjectMeta: metav1.ObjectMeta{Name: orgName, Namespace: clusterNamespace},
			})).To(Succeed())
		})

		It("should successfully reconcile the resource if all fields are valid and the cluster is healthy", func() {
			By("Reconciling the created resource")
			controllerReconciler := &controllerImpl.ClusterReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				TargetFactory: targetClientFactory,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// check the cluster resource
			cluster := &platformv1alpha1.Cluster{}
			err = k8sClient.Get(ctx, typeNamespacedName, cluster)
			Expect(err).NotTo(HaveOccurred())

			// condition := apimeta.FindStatusCondition(cluster.Status.Conditions, platformv1alpha1.Ready)
			// Expect(condition).NotTo(BeNil())
			// Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			// Expect(condition.Reason).To(Equal("Reconciled"))
			// Expect(condition.Message).To(Equal("Cluster is healthy"))

			// check the cluster ready condition
			Eventually(func(g Gomega) {
				var updated platformv1alpha1.Cluster
				err := k8sClient.Get(ctx, typeNamespacedName, &updated)
				g.Expect(err).ToNot(HaveOccurred())

				cond := apimeta.FindStatusCondition(updated.Status.Conditions, platformv1alpha1.Ready)
				g.Expect(cond).ToNot(BeNil())
				g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(cond.Reason).To(Equal("Reconciled"))
			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

			// check the kubeconfig secret
			kcfg := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: clusterSecretName, Namespace: clusterNamespace}, kcfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(kcfg.Data["kubeconfig"]).NotTo(BeEmpty())
		})

		It("marks Ready=False when the kubeconfig secret is missing", func() {
			ctx := context.Background()

			// create the Cluster but NOT its secret
			cluster := &platformv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-secret",
					Namespace: clusterNamespace,
				},
				Spec: platformv1alpha1.ClusterSpec{
					OrgRef:           orgName,
					Type:             clusterType,
					KubeconfigSecret: "idontexist",
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			r := &controllerImpl.ClusterReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				TargetFactory: targetClientFactory,
			}

			// trigger one reconcile cycle
			res, err := r.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				},
			})

			// assert the cluster is not ready
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(Equal(5 * time.Minute))

			Eventually(func(g Gomega) {
				var updated platformv1alpha1.Cluster
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &updated)).To(Succeed())

				ready := apimeta.FindStatusCondition(updated.Status.Conditions, platformv1alpha1.Ready)
				errC := apimeta.FindStatusCondition(updated.Status.Conditions, platformv1alpha1.Error)

				g.Expect(ready).NotTo(BeNil())
				g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(ready.Reason).To(Equal("KubeconfigSecretMissing"))

				g.Expect(errC).NotTo(BeNil())
				g.Expect(errC.Status).To(Equal(metav1.ConditionTrue))
			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
		})

		It("marks Ready=False when the cluster has zero nodes", func() {
			ctx := context.Background()

			cluster := &platformv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-nodes",
					Namespace: clusterNamespace,
				},
				Spec: platformv1alpha1.ClusterSpec{
					OrgRef:           orgName,
					Type:             clusterType,
					KubeconfigSecret: clusterSecretName,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			r := &controllerImpl.ClusterReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				TargetFactory: &noNodeTargetClusterClientFactory{},
			}

			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				var updated platformv1alpha1.Cluster
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &updated)).To(Succeed())

				ready := apimeta.FindStatusCondition(updated.Status.Conditions, platformv1alpha1.Ready)
				g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(ready.Reason).To(Equal("HealthCheckFailed"))
				g.Expect(ready.Message).To(ContainSubstring("No nodes found"))
			}).Should(Succeed())
		})

		It("marks Ready=False when at least one node is NotReady", func() {
			ctx := context.Background()

			// create a fake node for the test cluster
			Expect(k8sClient.Create(ctx, &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "sad-node"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{{
						Type:               corev1.NodeReady,
						Status:             corev1.ConditionFalse,
						LastHeartbeatTime:  metav1.Now(),
						LastTransitionTime: metav1.Now(),
						Reason:             "KubeletNotReady",
					}},
				},
			})).To(Succeed())

			// cluster object
			cluster := &platformv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unhealthy",
					Namespace: clusterNamespace,
				},
				Spec: platformv1alpha1.ClusterSpec{
					OrgRef:           orgName,
					Type:             clusterType,
					KubeconfigSecret: clusterSecretName,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			r := &controllerImpl.ClusterReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				TargetFactory: targetClientFactory, // real factory OK; env-test has that node
			}

			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				var updated platformv1alpha1.Cluster
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &updated)).To(Succeed())

				ready := apimeta.FindStatusCondition(updated.Status.Conditions, platformv1alpha1.Ready)
				g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(ready.Reason).To(Equal("HealthCheckFailed"))
				g.Expect(ready.Message).To(ContainSubstring("is not ready"))
			}).Should(Succeed())
		})

		It("updates cluster metrics when the cluster is created", func() {
			By("Reconciling the created resource")
			controllerReconciler := &controllerImpl.ClusterReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				TargetFactory: targetClientFactory,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				got := testutil.ToFloat64(metrics.ClustersPerOrg.WithLabelValues(orgName))
				g.Expect(got).To(BeNumerically("==", 1))
			}).Should(Succeed())
		})

		It("updates cluster metrics when the cluster is deleted", func() {
			//TODO: Implement this test
		})

		It("adds the finalizer to the cluster on creation", func() {
			//TODO: Implement this test
		})

		It("removes the finalizer from the cluster on deletion", func() {
			//TODO: Implement this test
		})
	})
})
