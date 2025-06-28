package controller

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	controllerImpl "github.com/mofe64/vulkan/operator/internal/controller"
	"github.com/mofe64/vulkan/operator/internal/metrics"
	utils "github.com/mofe64/vulkan/operator/internal/utils"
	testUtils "github.com/mofe64/vulkan/operator/test/utils"
)

// -----------------------------------------------------------------------------
// Helpers shared by all specs
// -----------------------------------------------------------------------------

// Target‑factory stub that always returns an *empty* cluster (zero Nodes).
// Used by the "no nodes" scenario.
type noNodeTargetClusterClientFactory struct{}

func (f *noNodeTargetClusterClientFactory) ClientFor(ctx context.Context, _ *platformv1alpha1.Cluster) (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).Build(), nil
}

func buildTestClusterReconciler(targetClientFactory utils.TargetClientFactory) *controllerImpl.ClusterReconciler {
	return &controllerImpl.ClusterReconciler{
		Client:        k8sClient,
		Scheme:        k8sClient.Scheme(),
		TargetFactory: targetClientFactory,
	}
}

func resetMetrics() {
	metrics.ClustersPerOrg.Reset()
	metrics.ProjectsPerOrg.Reset()
	metrics.ApplicationsPerOrg.Reset()
	metrics.OrgQuotaUsage.Reset()
}

// makeCluster scaffolds a Cluster CR for the given namespace / secret.
func makeCluster(ns, secretName, orgID string, clusterType string) *platformv1alpha1.Cluster {
	return &platformv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-" + uuid.NewString(),
			Namespace: ns,
		},
		Spec: platformv1alpha1.ClusterSpec{
			OrgRef:                    orgID,
			Type:                      clusterType,
			KubeconfigSecretName:      secretName,
			KubeconfigSecretNamespace: ns,
			DisplayName:               "display‑" + uuid.NewString(),
			ClusterID:                 uuid.NewString(),
		},
	}
}

func makeOrg(ns, orgID string) *platformv1alpha1.Org {
	return &platformv1alpha1.Org{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org-" + orgID,
			Namespace: ns,
		},
		Spec: platformv1alpha1.OrgSpec{
			OrgID:       orgID,
			DisplayName: "display-" + uuid.NewString(),
			OwnerEmail:  "test@test.com",
			OrgQuota:    platformv1alpha1.OrgQuota{Clusters: 10, Apps: 10},
		},
	}
}

func makeOrgWithQuota(ns, orgID string, clusters int32, apps int32) *platformv1alpha1.Org {
	return &platformv1alpha1.Org{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org-" + orgID,
			Namespace: ns,
		},
		Spec: platformv1alpha1.OrgSpec{
			OrgID:       orgID,
			DisplayName: "display-" + uuid.NewString(),
			OwnerEmail:  "test@test.com",
			OrgQuota:    platformv1alpha1.OrgQuota{Clusters: clusters, Apps: apps},
		},
	}
}

var _ = Describe("Cluster controller", Ordered, Serial, func() {
	var (
		ctx context.Context

		// per‑spec resources
		ns           *corev1.Namespace
		kubeSecret   *corev1.Secret
		reconciler   *controllerImpl.ClusterReconciler
		createdNodes []string // cluster‑wide objects have to be cleaned up explicitly
	)

	BeforeEach(func() {
		ctx = context.Background()
		createdNodes = nil

		// isolated namespace
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "test-" + uuid.NewString(),
		}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		// kubeconfig secret
		kcBytes, err := testUtils.KubeconfigWithEmbeddedCA(testEnv.Config)
		Expect(err).NotTo(HaveOccurred())

		kubeSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "kubeconfig-" + uuid.NewString(),
			},
			Data: map[string][]byte{"kubeconfig": kcBytes},
		}
		Expect(k8sClient.Create(ctx, kubeSecret)).To(Succeed())

		// reconciler & fresh metrics
		reconciler = buildTestClusterReconciler(utils.NewTargetClientFactory(k8sClient))
		resetMetrics()
	})

	AfterEach(func() {
		// delete any cluster‑scoped objects (Nodes) that this spec created
		for _, n := range createdNodes {
			_ = k8sClient.Delete(ctx, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: n}})
		}
		// finally, drop the namespace – this removes *all* namespaced objects.
		_ = k8sClient.Delete(ctx, ns)
	})

	// happy path – Ready=True
	It("sets Ready=True when the cluster is healthy", func() {
		// fake a Ready node so health‑check passes
		readyNodeName := "ready-" + uuid.NewString()
		createdNodes = append(createdNodes, readyNodeName)
		Expect(k8sClient.Create(ctx, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: readyNodeName},
			Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
				Type:               corev1.NodeReady,
				Status:             corev1.ConditionTrue,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			}}},
		})).To(Succeed())

		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		// trigger reconcile once
		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: cluster.Namespace,
				Name:      cluster.Name,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		// Assert Ready condition becomes True
		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())
			cond := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
	})

	// error path – kubeconfig secret missing
	It("sets Ready=False when the kubeconfig secret is missing", func() {
		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())
		By("creating the cluster resource with an invalid kubeconfigsecret for the test Cluster")
		cluster := makeCluster(ns.Name, "idontexist", org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: cluster.Namespace,
				Name:      cluster.Name,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())

			ready := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(ready.Reason).To(Equal("KubeconfigSecretMissing"))
		}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
	})

	// error path – zero nodes in the attached target cluster
	It("sets Ready=False when the cluster has zero nodes and cluster is attached", func() {
		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: cluster.Namespace,
				Name:      cluster.Name,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())
			ready := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(ready.Reason).To(Equal("HealthCheckFailed"))
			g.Expect(ready.Message).To(ContainSubstring("No nodes"))
		}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).Should(Succeed())
	})

	// error path – zero nodes in the remote target cluster
	It("sets Ready=False when the cluster has zero nodes and cluster is remote", func() {
		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "remote")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		// Use a factory that returns an empty client (no Nodes)
		r := buildTestClusterReconciler(&noNodeTargetClusterClientFactory{})
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())
			ready := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(ready.Reason).To(Equal("HealthCheckFailed"))
			g.Expect(ready.Message).To(ContainSubstring("No nodes"))
		}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).Should(Succeed())
	})

	// error path – at least one NotReady node
	It("sets Ready=False when a node is NotReady", func() {
		notReadyNode := "sad-" + uuid.NewString()
		createdNodes = append(createdNodes, notReadyNode)
		Expect(k8sClient.Create(ctx, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: notReadyNode},
			Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
				Type:               corev1.NodeReady,
				Status:             corev1.ConditionFalse,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Reason:             "KubeletNotReady",
			}}},
		})).To(Succeed())

		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())
			ready := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(ready.Reason).To(Equal("HealthCheckFailed"))
			g.Expect(ready.Message).To(ContainSubstring("not ready"))
		}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).Should(Succeed())
	})

	// metrics – ensure they increment / decrement correctly

	It("updates cluster metrics on create & delete", func() {
		// fake a Ready node so health‑check passes
		readyNodeName := "ready-" + uuid.NewString()
		createdNodes = append(createdNodes, readyNodeName)
		Expect(k8sClient.Create(ctx, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: readyNodeName},
			Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
				Type:               corev1.NodeReady,
				Status:             corev1.ConditionTrue,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			}}},
		})).To(Succeed())

		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		// first reconcile – metrics should go to 1
		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKey{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			g.Expect(testutil.ToFloat64(metrics.ClustersPerOrg.WithLabelValues(cluster.Spec.OrgRef))).
				To(BeNumerically("==", 1))
		}).Should(Succeed())

		// delete cluster & reconcile again – metrics should drop to 0
		Expect(k8sClient.Delete(ctx, cluster)).To(Succeed())
		_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			g.Expect(testutil.ToFloat64(metrics.ClustersPerOrg.WithLabelValues(cluster.Spec.OrgRef))).
				To(BeNumerically("==", 0))
		}).Should(Succeed())
	})

	// finalizer – added on create
	It("adds the finalizer to the Cluster on creation", func() {
		// fake a Ready node so health‑check passes
		readyNodeName := "ready-" + uuid.NewString()
		createdNodes = append(createdNodes, readyNodeName)
		Expect(k8sClient.Create(ctx, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: readyNodeName},
			Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
				Type:               corev1.NodeReady,
				Status:             corev1.ConditionTrue,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			}}},
		})).To(Succeed())
		org := makeOrg(ns.Name, uuid.NewString())
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())
			g.Expect(got.Finalizers).To(ContainElement(platformv1alpha1.ClusterFinalizer))
		}).Should(Succeed())
	})

	// validate org quota
	It("sets Ready=False and Error=True when the org quota is exceeded", func() {
		// fake a Ready node so health‑check passes
		readyNodeName := "ready-" + uuid.NewString()
		createdNodes = append(createdNodes, readyNodeName)
		Expect(k8sClient.Create(ctx, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: readyNodeName},
			Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
				Type:               corev1.NodeReady,
				Status:             corev1.ConditionTrue,
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
			}}},
		})).To(Succeed())

		org := makeOrgWithQuota(ns.Name, uuid.NewString(), 1, 1)
		Expect(k8sClient.Create(ctx, org)).To(Succeed())

		cluster := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

		// trigger reconcile once
		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: cluster.Namespace,
				Name:      cluster.Name,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		// Assert Ready condition becomes True
		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), &got)).To(Succeed())
			cond := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

		// make second cluster
		cluster2 := makeCluster(ns.Name, kubeSecret.Name, org.Spec.OrgID, "attached")
		Expect(k8sClient.Create(ctx, cluster2)).To(Succeed())

		_, err = reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: cluster2.Namespace,
				Name:      cluster2.Name,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			var got platformv1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster2), &got)).To(Succeed())
			ready := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Ready)
			errorCondition := apimeta.FindStatusCondition(got.Status.Conditions, platformv1alpha1.Error)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(errorCondition).NotTo(BeNil())
			g.Expect(errorCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(errorCondition.Reason).To(Equal("ClusterQuotaExceeded"))
		}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
	})
})
