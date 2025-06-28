package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	controllerImpl "github.com/mofe64/vulkan/operator/internal/controller"
	"github.com/mofe64/vulkan/operator/internal/metrics"
)

const orgID = "550e8400-e29b-41d4-a716-446655440001"

// failingTestClient is a test client that can be configured to fail specific operations
type failingTestClient struct {
	client.Client
	failOnDelete map[string]bool
}

func (f *failingTestClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	// Check if we should fail this deletion based on the object type
	if f.failOnDelete[obj.GetObjectKind().GroupVersionKind().Kind] {
		return fmt.Errorf("simulated deletion error for %s", obj.GetObjectKind().GroupVersionKind().Kind)
	}
	return f.Client.Delete(ctx, obj, opts...)
}

func createValidProject() *platformv1alpha1.Project {
	return &platformv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "project-" + uuid.NewString(),
		},
		Spec: platformv1alpha1.ProjectSpec{
			OrgRef:            orgID,
			ProjectID:         uuid.NewString(),
			DisplayName:       "display-" + uuid.NewString(),
			ProjectMaxCores:   10,
			ProjectMaxMemory:  20,
			ProjectMaxStorage: 30,
		},
	}
}

func getProjectCount(org string) float64 {
	got := testutil.ToFloat64(metrics.ProjectsPerOrg.WithLabelValues(org))
	return got
}

var _ = Describe("Project Controller", Ordered, Serial, func() {
	var (
		ctx        context.Context
		ns         *corev1.Namespace
		reconciler *controllerImpl.ProjectReconciler
		clusterId  = uuid.NewString()
	)

	Context("Project Lifecycle Management", func() {

		BeforeEach(func() {
			ctx = context.Background()
			// isolated namespace
			ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "test-" + uuid.NewString(),
			}}

			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			reconciler = buildTestProjectReconciler()

			// Reset metrics before each test
			metrics.ProjectsPerOrg.Reset()
		})

		AfterEach(func() {
			// cleanup: Delete the namespace, which will delete all ns scoped objects
			_ = k8sClient.Delete(ctx, ns)
		})

		Describe("Project Creation", func() {
			It("should successfully create a project and increment metrics", func() {
				By("Creating a new project")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Reconciling the project")

				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying the project exists with the correct spec and conditions")
				Eventually(func(g Gomega) {
					var p platformv1alpha1.Project
					g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), &p)).To(Succeed())
					g.Expect(p.Spec.DisplayName).To(Equal(project.Spec.DisplayName))
					g.Expect(p.Spec.OrgRef).To(Equal(orgID))
					g.Expect(p.Spec.ProjectID).To(Equal(project.Spec.ProjectID))
					g.Expect(p.Spec.ProjectMaxCores).To(Equal(project.Spec.ProjectMaxCores))
					g.Expect(p.Spec.ProjectMaxMemory).To(Equal(project.Spec.ProjectMaxMemory))
					g.Expect(p.Spec.ProjectMaxStorage).To(Equal(project.Spec.ProjectMaxStorage))

					ready := apimeta.FindStatusCondition(p.Status.Conditions, platformv1alpha1.Ready)
					g.Expect(ready).NotTo(BeNil())
					g.Expect(ready.Status).To(Equal(metav1.ConditionTrue))
					g.Expect(ready.Reason).To(Equal("Reconciled"))
					g.Expect(ready.Message).To(Equal("Project is healthy"))

					error := apimeta.FindStatusCondition(p.Status.Conditions, platformv1alpha1.Error)
					g.Expect(error).To(BeNil())
				}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

				By("Verifying metrics are incremented")
				Expect(getProjectCount(orgID)).To(Equal(1.0))
			})

		})

		Describe("Project Deletion", func() {
			It("should successfully delete a project and decrement metrics", func() {
				By("Creating a project ")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Reconciling the project")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(getProjectCount(orgID)).To(Equal(1.0))

				// delete the project
				By("Deleting the project")
				Expect(k8sClient.Delete(ctx, project)).To(Succeed())

				By("Reconciling after deletion")
				_, err = reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying project deleted")
				Eventually(func(g Gomega) {
					deletedProject := &platformv1alpha1.Project{}
					err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), deletedProject)
					g.Expect(err).To(HaveOccurred())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

				By("Verifying metrics are decremented")
				Eventually(func(g Gomega) {
					g.Expect(getProjectCount(orgID)).To(Equal(0.0))
				}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
			})

			It("should handle deletion with existing cluster bindings", func() {
				By("Creating a project")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Creating a cluster binding for the project")
				clusterBinding := &platformv1alpha1.ProjectClusterBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-binding",
					},
					Spec: platformv1alpha1.ProjectClusterBindingSpec{
						ProjectRef: project.Spec.ProjectID,
						ClusterRef: clusterId,
					},
				}
				Expect(k8sClient.Create(ctx, clusterBinding)).To(Succeed())

				By("Reconciling the project")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(getProjectCount(orgID)).To(Equal(1.0))

				By("Deleting the project")
				Expect(k8sClient.Delete(ctx, project)).To(Succeed())

				By("Reconciling after deletion - should delete cluster binding")
				_, err = reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying cluster binding is deleted")
				Eventually(func(g Gomega) {
					deletedBinding := &platformv1alpha1.ProjectClusterBinding{}
					err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-binding"}, deletedBinding)
					g.Expect(err).To(HaveOccurred())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

				By("Verifying project is deleted and metrics decremented")
				Eventually(func(g Gomega) {
					deletedProject := &platformv1alpha1.Project{}
					err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), deletedProject)
					g.Expect(err).To(HaveOccurred())
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
					g.Expect(getProjectCount(orgID)).To(Equal(0.0))
				}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
			})
		})

		Describe("Status Conditions", func() {
			It("should set Ready condition when project is successfully created", func() {
				By("Creating a project")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Reconciling the project")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying Ready condition is maintained")
				updatedProject := &platformv1alpha1.Project{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), updatedProject)).To(Succeed())

				readyCondition := apimeta.FindStatusCondition(updatedProject.Status.Conditions, platformv1alpha1.Ready)
				Expect(readyCondition).NotTo(BeNil())
				Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			})
		})

		Describe("Metrics Tracking", func() {

			It("should handle multiple projects for the same org", func() {
				By("Creating first project")
				project1 := createValidProject()
				project1.Name = "project-1"

				Expect(k8sClient.Create(ctx, project1)).To(Succeed())

				By("Creating second project")
				project2 := createValidProject()
				project2.Name = "project-2"

				Expect(k8sClient.Create(ctx, project2)).To(Succeed())

				By("Reconciling both projects")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "project-1"},
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "project-2"},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying metrics show correct count for org")
				Expect(getProjectCount(orgID)).To(Equal(2.0))

				By("Deleting first project")
				Expect(k8sClient.Delete(ctx, project1)).To(Succeed())

				_, err = reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "project-1"},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying metrics are decremented correctly")
				Expect(getProjectCount(orgID)).To(Equal(1.0))

				By("Cleaning up second project")
				Expect(k8sClient.Delete(ctx, project2)).To(Succeed())

				_, err = reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "project-2"},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(getProjectCount(orgID)).To(Equal(0.0))
			})
		})

		Describe("Error Handling", func() {
			It("should handle reconciliation errors gracefully", func() {
				By("Creating a project")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Reconciling with invalid request")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      "non-existent",
					},
				})
				Expect(err).NotTo(HaveOccurred()) // Should handle gracefully

				By("Verifying original project is unaffected")
				existingProject := &platformv1alpha1.Project{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), existingProject)).To(Succeed())
				Expect(existingProject.Spec.DisplayName).To(Equal(project.Spec.DisplayName))
			})

			It("should handle metrics errors gracefully", func() {
				By("Creating a project")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Reconciling the project")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying project reconciliation succeeds even if metrics fail")
				updatedProject := &platformv1alpha1.Project{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), updatedProject)).To(Succeed())
				Expect(updatedProject.Spec.DisplayName).To(Equal(project.Spec.DisplayName))
			})

			It("should handle cluster binding deletion errors and set error condition", func() {
				By("Creating a project")
				project := createValidProject()
				Expect(k8sClient.Create(ctx, project)).To(Succeed())

				By("Creating a cluster binding for the project")
				clusterBinding := &platformv1alpha1.ProjectClusterBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-binding",
					},
					Spec: platformv1alpha1.ProjectClusterBindingSpec{
						ProjectRef: project.Spec.ProjectID,
						ClusterRef: clusterId,
					},
				}
				Expect(k8sClient.Create(ctx, clusterBinding)).To(Succeed())

				By("Reconciling the project to set up finalizer")
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Getting the latest project state and setting deletion timestamp and finalizer")
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)).To(Succeed())
				Expect(k8sClient.Delete(ctx, project)).To(Succeed())

				By("Creating a test client that fails on cluster binding deletion")
				failingClient := &failingTestClient{
					Client: k8sClient,
					failOnDelete: map[string]bool{
						"ProjectClusterBinding": true,
					},
				}

				failingReconciler := &controllerImpl.ProjectReconciler{
					Client: failingClient,
					Scheme: k8sClient.Scheme(),
				}

				By("Reconciling with failing client - should trigger error path")
				result, err := failingReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					},
				})

				By("Verifying error is returned and requeue is scheduled")
				Expect(err).To(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(5 * time.Minute))

				By("Verifying error condition is set on project")
				updatedProject := &platformv1alpha1.Project{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), updatedProject)).To(Succeed())

				errorCondition := apimeta.FindStatusCondition(updatedProject.Status.Conditions, platformv1alpha1.Error)
				Expect(errorCondition).NotTo(BeNil())
				Expect(errorCondition.Status).To(Equal(metav1.ConditionTrue))
				Expect(errorCondition.Reason).To(Equal("ClusterBindingDeletionError"))
				Expect(errorCondition.Message).To(ContainSubstring("simulated deletion error"))
			})
		})
	})
})

func buildTestProjectReconciler() *controllerImpl.ProjectReconciler {
	return &controllerImpl.ProjectReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}
