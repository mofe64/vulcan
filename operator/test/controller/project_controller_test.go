package controller

// import (
// 	"context"
// 	"time"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	"github.com/prometheus/client_golang/prometheus/testutil"
// 	"k8s.io/apimachinery/pkg/api/errors"
// 	"k8s.io/apimachinery/pkg/types"
// 	"sigs.k8s.io/controller-runtime/pkg/reconcile"

// 	apimeta "k8s.io/apimachinery/pkg/api/meta"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// 	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
// 	"github.com/mofe64/vulkan/operator/internal/controller"
// 	"github.com/mofe64/vulkan/operator/internal/metrics"
// )

// var _ = Describe("Project Controller", func() {
// 	Context("Project Lifecycle Management", func() {
// 		const (
// 			projectName = "test-project"
// 			orgRef      = "550e8400-e29b-41d4-a716-446655440001"
// 			projectID   = "550e8400-e29b-41d4-a716-446655440002"
// 		)

// 		ctx := context.Background()
// 		typeNamespacedName := types.NamespacedName{
// 			Name: projectName,
// 		}

// 		// Helper function to create a valid project
// 		createValidProject := func() *platformv1alpha1.Project {
// 			return &platformv1alpha1.Project{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: projectName,
// 				},
// 				Spec: platformv1alpha1.ProjectSpec{
// 					OrgRef:            orgRef,
// 					ProjectID:         projectID,
// 					DisplayName:       "Test Project",
// 					ProjectMaxCores:   10,
// 					ProjectMaxMemory:  20,
// 					ProjectMaxStorage: 30,
// 				},
// 			}
// 		}

// 		// Helper function to get current project count for an org
// 		getProjectCount := func(org string) float64 {
// 			got := testutil.ToFloat64(metrics.ClustersPerOrg.WithLabelValues(org))

// 			return got
// 		}

// 		BeforeEach(func() {
// 			// Reset metrics before each test
// 			metrics.ProjectsPerOrg.Reset()
// 		})

// 		AfterEach(func() {
// 			// Cleanup: Delete the project if it exists
// 			project := &platformv1alpha1.Project{}
// 			err := k8sClient.Get(ctx, typeNamespacedName, project)
// 			if err == nil {
// 				// Force delete by removing finalizers
// 				project.Finalizers = []string{}
// 				Expect(k8sClient.Update(ctx, project)).To(Succeed())
// 				Expect(k8sClient.Delete(ctx, project)).To(Succeed())

// 				// Wait for the project to be fully deleted
// 				Eventually(func(g Gomega) {
// 					err := k8sClient.Get(ctx, typeNamespacedName, project)
// 					g.Expect(err).To(HaveOccurred())
// 					g.Expect(errors.IsNotFound(err)).To(BeTrue())
// 				}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
// 			}
// 		})

// 		Describe("Project Creation", func() {
// 			It("should successfully create a project and increment metrics", func() {
// 				By("Creating a new project")
// 				project := createValidProject()
// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling the project")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}

// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying the project exists")
// 				createdProject := &platformv1alpha1.Project{}
// 				Expect(k8sClient.Get(ctx, typeNamespacedName, createdProject)).To(Succeed())
// 				Expect(createdProject.Spec.DisplayName).To(Equal("Test Project"))
// 				Expect(createdProject.Spec.OrgRef).To(Equal(orgRef))

// 				By("Verifying metrics are incremented")
// 				Expect(getProjectCount(orgRef)).To(Equal(1.0))
// 			})

// 			It("should set Ready condition and increment metrics when project is ready", func() {
// 				By("Creating a project with Ready condition")
// 				project := createValidProject()

// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling the project")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}

// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying metrics are incremented for ready project")
// 				Expect(getProjectCount(orgRef)).To(Equal(1.0))
// 			})
// 		})

// 		Describe("Project Deletion", func() {
// 			It("should successfully delete a project and decrement metrics", func() {
// 				By("Creating a project ")
// 				project := createValidProject()
// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling to increment metrics")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}
// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())
// 				Expect(getProjectCount(orgRef)).To(Equal(1.0))

// 				By("Marking project for deletion with finalizer")
// 				project.Finalizers = []string{platformv1alpha1.ProjectFinalizer}
// 				project.DeletionTimestamp = &metav1.Time{Time: time.Now()}
// 				Expect(k8sClient.Update(ctx, project)).To(Succeed())

// 				By("Reconciling during deletion")
// 				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying finalizer is removed")
// 				deletedProject := &platformv1alpha1.Project{}
// 				err = k8sClient.Get(ctx, typeNamespacedName, deletedProject)
// 				Expect(err).To(HaveOccurred())
// 				Expect(errors.IsNotFound(err)).To(BeTrue())

// 				By("Verifying metrics are decremented")
// 				Expect(getProjectCount(orgRef)).To(Equal(0.0))
// 			})

// 			It("should handle deletion with existing cluster bindings", func() {
// 				By("Creating a project")
// 				project := createValidProject()
// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Creating a cluster binding for the project")
// 				clusterBinding := &platformv1alpha1.ProjectClusterBinding{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "test-binding",
// 					},
// 					Spec: platformv1alpha1.ProjectClusterBindingSpec{
// 						ProjectRef: projectName,
// 						ClusterRef: "test-cluster",
// 					},
// 				}
// 				Expect(k8sClient.Create(ctx, clusterBinding)).To(Succeed())

// 				By("Reconciling to increment metrics")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}
// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())
// 				Expect(getProjectCount(orgRef)).To(Equal(1.0))

// 				By("Marking project for deletion")
// 				project.Finalizers = []string{platformv1alpha1.ProjectFinalizer}
// 				project.DeletionTimestamp = &metav1.Time{Time: time.Now()}
// 				Expect(k8sClient.Update(ctx, project)).To(Succeed())

// 				By("Reconciling during deletion - should delete cluster binding")
// 				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying cluster binding is deleted")
// 				deletedBinding := &platformv1alpha1.ProjectClusterBinding{}
// 				err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-binding"}, deletedBinding)
// 				Expect(err).To(HaveOccurred())
// 				Expect(errors.IsNotFound(err)).To(BeTrue())

// 				By("Verifying project is deleted and metrics decremented")
// 				deletedProject := &platformv1alpha1.Project{}
// 				err = k8sClient.Get(ctx, typeNamespacedName, deletedProject)
// 				Expect(err).To(HaveOccurred())
// 				Expect(errors.IsNotFound(err)).To(BeTrue())
// 				Expect(getProjectCount(orgRef)).To(Equal(0.0))
// 			})

// 			It("should handle cluster binding deletion errors gracefully", func() {
// 				By("Creating a project with Ready condition")
// 				project := createValidProject()

// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling to increment metrics")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}
// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())
// 				Expect(getProjectCount(orgRef)).To(Equal(1.0))

// 				By("Marking project for deletion")
// 				project.Finalizers = []string{platformv1alpha1.ProjectFinalizer}
// 				project.DeletionTimestamp = &metav1.Time{Time: time.Now()}
// 				Expect(k8sClient.Update(ctx, project)).To(Succeed())

// 				By("Reconciling during deletion")
// 				result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying requeue is scheduled for error handling")
// 				Expect(result.RequeueAfter).To(Equal(5 * time.Minute))

// 				By("Verifying error condition is set")
// 				deletedProject := &platformv1alpha1.Project{}
// 				Expect(k8sClient.Get(ctx, typeNamespacedName, deletedProject)).To(Succeed())

// 				errorCondition := apimeta.FindStatusCondition(deletedProject.Status.Conditions, platformv1alpha1.Error)
// 				Expect(errorCondition).NotTo(BeNil())
// 				Expect(errorCondition.Status).To(Equal(metav1.ConditionTrue))
// 				Expect(errorCondition.Reason).To(Equal("ClusterBindingDeletionError"))
// 			})
// 		})

// 		Describe("Status Conditions", func() {
// 			It("should set Ready condition when project is successfully created", func() {
// 				By("Creating a project")
// 				project := createValidProject()
// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling the project")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}

// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying Ready condition is maintained")
// 				updatedProject := &platformv1alpha1.Project{}
// 				Expect(k8sClient.Get(ctx, typeNamespacedName, updatedProject)).To(Succeed())

// 				readyCondition := apimeta.FindStatusCondition(updatedProject.Status.Conditions, platformv1alpha1.Ready)
// 				Expect(readyCondition).NotTo(BeNil())
// 				Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
// 			})
// 		})

// 		Describe("Metrics Tracking", func() {

// 			It("should handle multiple projects for the same org", func() {
// 				By("Creating first project with Ready condition")
// 				project1 := createValidProject()
// 				project1.Name = "project-1"
// 				project1.Status.Conditions = []metav1.Condition{
// 					{
// 						Type:   platformv1alpha1.Ready,
// 						Status: metav1.ConditionTrue,
// 						Reason: "ProjectReady",
// 					},
// 				}
// 				Expect(k8sClient.Create(ctx, project1)).To(Succeed())

// 				By("Creating second project with Ready condition")
// 				project2 := createValidProject()
// 				project2.Name = "project-2"
// 				project2.Status.Conditions = []metav1.Condition{
// 					{
// 						Type:   platformv1alpha1.Ready,
// 						Status: metav1.ConditionTrue,
// 						Reason: "ProjectReady",
// 					},
// 				}
// 				Expect(k8sClient.Create(ctx, project2)).To(Succeed())

// 				By("Reconciling both projects")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}

// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: types.NamespacedName{Name: "project-1"},
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: types.NamespacedName{Name: "project-2"},
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying metrics show correct count for org")
// 				Expect(getProjectCount(orgRef)).To(Equal(2.0))

// 				By("Deleting first project")
// 				project1.Finalizers = []string{platformv1alpha1.ProjectFinalizer}
// 				project1.DeletionTimestamp = &metav1.Time{Time: time.Now()}
// 				Expect(k8sClient.Update(ctx, project1)).To(Succeed())

// 				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: types.NamespacedName{Name: "project-1"},
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying metrics are decremented correctly")
// 				Expect(getProjectCount(orgRef)).To(Equal(1.0))

// 				By("Cleaning up second project")
// 				project2.Finalizers = []string{platformv1alpha1.ProjectFinalizer}
// 				project2.DeletionTimestamp = &metav1.Time{Time: time.Now()}
// 				Expect(k8sClient.Update(ctx, project2)).To(Succeed())

// 				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: types.NamespacedName{Name: "project-2"},
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				Expect(getProjectCount(orgRef)).To(Equal(0.0))
// 			})
// 		})

// 		Describe("Error Handling", func() {
// 			It("should handle reconciliation errors gracefully", func() {
// 				By("Creating a project")
// 				project := createValidProject()
// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling with invalid request")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}

// 				// Test with non-existent project
// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: types.NamespacedName{Name: "non-existent"},
// 				})
// 				Expect(err).NotTo(HaveOccurred()) // Should handle gracefully

// 				By("Verifying original project is unaffected")
// 				existingProject := &platformv1alpha1.Project{}
// 				Expect(k8sClient.Get(ctx, typeNamespacedName, existingProject)).To(Succeed())
// 				Expect(existingProject.Spec.DisplayName).To(Equal("Test Project"))
// 			})

// 			It("should handle metrics errors gracefully", func() {
// 				By("Creating a project")
// 				project := createValidProject()
// 				Expect(k8sClient.Create(ctx, project)).To(Succeed())

// 				By("Reconciling the project")
// 				controllerReconciler := &controller.ProjectReconciler{
// 					Client: k8sClient,
// 					Scheme: k8sClient.Scheme(),
// 				}

// 				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 					NamespacedName: typeNamespacedName,
// 				})
// 				Expect(err).NotTo(HaveOccurred())

// 				By("Verifying project reconciliation succeeds even if metrics fail")
// 				updatedProject := &platformv1alpha1.Project{}
// 				Expect(k8sClient.Get(ctx, typeNamespacedName, updatedProject)).To(Succeed())
// 				Expect(updatedProject.Spec.DisplayName).To(Equal("Test Project"))
// 			})
// 		})
// 	})
// })
