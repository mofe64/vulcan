package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	controllerImpl "github.com/mofe64/vulkan/operator/internal/controller"
	model "github.com/mofe64/vulkan/operator/internal/model"
	utils "github.com/mofe64/vulkan/operator/internal/utils"
)

func makeClusterForCBTest(ns, secretName, orgRef string, clusterType string) *platformv1alpha1.Cluster {
	return &platformv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-" + uuid.NewString(),
			Namespace: ns,
		},
		Spec: platformv1alpha1.ClusterSpec{
			OrgRef:                    orgRef,
			Type:                      clusterType,
			KubeconfigSecretName:      secretName,
			KubeconfigSecretNamespace: ns,
			DisplayName:               "display‑" + uuid.NewString(),
			ClusterID:                 uuid.NewString(),
		},
	}
}

func makeProjectForCBTest(ns, orgRef string) *platformv1alpha1.Project {
	return &platformv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-" + uuid.NewString(),
			Namespace: ns,
		},
		Spec: platformv1alpha1.ProjectSpec{
			OrgRef:            orgRef,
			ProjectID:         uuid.NewString(),
			DisplayName:       "display-" + uuid.NewString(),
			ProjectMaxCores:   10,
			ProjectMaxMemory:  20,
			ProjectMaxStorage: 30,
		},
	}
}

func makeProjectForCBTestWithProjectNamespace(ns, orgRef, projectNamespace string) *platformv1alpha1.Project {
	return &platformv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-" + uuid.NewString(),
			Namespace: ns,
		},
		Spec: platformv1alpha1.ProjectSpec{
			OrgRef:            orgRef,
			ProjectID:         uuid.NewString(),
			DisplayName:       "display-" + uuid.NewString(),
			ProjectMaxCores:   10,
			ProjectMaxMemory:  20,
			ProjectMaxStorage: 30,
			ProjectNamespace:  utils.ShortName(projectNamespace, uuid.NewString()),
		},
	}
}

func makeOrgForCBTest(ns, orgID string) *platformv1alpha1.Org {
	return &platformv1alpha1.Org{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org-" + uuid.NewString(),
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

func makeProjectClusterBinding(ns, projectName, clusterName string) *platformv1alpha1.ProjectClusterBinding {
	return &platformv1alpha1.ProjectClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "projectclusterbinding-" + uuid.NewString(),
			Namespace: ns,
		},
		Spec: platformv1alpha1.ProjectClusterBindingSpec{
			ProjectRef: projectName,
			ClusterRef: clusterName,
		},
	}
}

var _ = Describe("ProjectClusterBinding Controller", func() {
	Context("When reconciling a resource", func() {

		var (
			ctx                  context.Context
			cbNamespace          *corev1.Namespace
			reconciler           *controllerImpl.ProjectClusterBindingReconciler
			org                  *platformv1alpha1.Org
			project              *platformv1alpha1.Project
			projectWithNamespace *platformv1alpha1.Project
			cluster              *platformv1alpha1.Cluster
			user1                model.ProjectMember
			user2                model.ProjectMember
			user3                model.ProjectMember
		)

		BeforeEach(func() {
			By("BeforeEach")
			ctx = context.Background()

			// isolated namespace
			cbNamespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "test-" + uuid.NewString(),
			}}
			Expect(k8sClient.Create(ctx, cbNamespace)).To(Succeed())

			reconciler = buildTestProjectClusterBindingReconciler()

			org = makeOrgForCBTest(cbNamespace.Name, uuid.NewString())
			Expect(k8sClient.Create(ctx, org)).To(Succeed())

			project = makeProjectForCBTest(cbNamespace.Name, org.Name)
			Expect(k8sClient.Create(ctx, project)).To(Succeed())

			projectWithNamespace = makeProjectForCBTestWithProjectNamespace(cbNamespace.Name, org.Name, "valid-namespace")
			Expect(k8sClient.Create(ctx, projectWithNamespace)).To(Succeed())

			cluster = makeClusterForCBTest(cbNamespace.Name, uuid.NewString(), org.Name, "attached")
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			user1 = model.ProjectMember{
				UserID:    uuid.NewString(),
				ProjectID: uuid.NewString(),
				Role:      "admin",
				Email:     "test1@test.com",
			}
			user2 = model.ProjectMember{
				UserID:    uuid.NewString(),
				ProjectID: uuid.NewString(),
				Role:      "maintainer",
				Email:     "test2@test.com",
			}
			user3 = model.ProjectMember{
				UserID:    uuid.NewString(),
				ProjectID: uuid.NewString(),
				Role:      "viewer",
				Email:     "test3@test.com",
			}

			_, err := testDB.ExecContext(
				ctx,
				`INSERT INTO users (id, oidc_sub, email, created_at) VALUES 
				($1, $2, $3, $4),
				($5, $6, $7, $8),
				($9, $10, $11, $12)`,
				user1.UserID, "user1_sub", user1.Email, time.Now().Format(time.RFC3339),
				user2.UserID, "user2_sub", user2.Email, time.Now().Format(time.RFC3339),
				user3.UserID, "user3_sub", user3.Email, time.Now().Format(time.RFC3339),
			)
			Expect(err).NotTo(HaveOccurred())

			_, err = testDB.ExecContext(ctx, `
				INSERT INTO projects (id, org_id, name, created_at) VALUES ($1, $2, $3, $4)
			`, project.Spec.ProjectID, org.Spec.OrgID, project.Name, time.Now().Format(time.RFC3339))
			Expect(err).NotTo(HaveOccurred())

			_, err = testDB.ExecContext(
				ctx, `INSERT INTO project_members (user_id, project_id, role) VALUES 
				($1, $2, $3),
				($4, $5, $6),
				($7, $8, $9)`,
				user1.UserID, project.Spec.ProjectID, user1.Role,
				user2.UserID, project.Spec.ProjectID, user2.Role,
				user3.UserID, project.Spec.ProjectID, user3.Role,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("AfterEach")
			Expect(k8sClient.Delete(ctx, cbNamespace)).To(Succeed())

			// delete all rows from all tables
			_, err := testDB.ExecContext(ctx, `DELETE FROM users`)
			Expect(err).NotTo(HaveOccurred())
			_, err = testDB.ExecContext(ctx, `DELETE FROM projects`)
			Expect(err).NotTo(HaveOccurred())
			_, err = testDB.ExecContext(ctx, `DELETE FROM project_members`)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should successfully reconcile the resource", func() {
			By("Creating a project cluster binding")
			binding := makeProjectClusterBinding(cbNamespace.Name, project.Name, cluster.Name)
			Expect(k8sClient.Create(ctx, binding)).To(Succeed())

			By("Reconciling the created resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      binding.Name,
					Namespace: cbNamespace.Name,
				},
			})

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a namespace for the project using the supplied project namespace if provided", func() {
			By("Creating a project cluster binding")
			binding := makeProjectClusterBinding(cbNamespace.Name, projectWithNamespace.Name, cluster.Name)
			Expect(k8sClient.Create(ctx, binding)).To(Succeed())

			By("Reconciling the created resource")
			reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      binding.Name,
					Namespace: cbNamespace.Name,
				},
			})

			By("Confirming that the namespace was created")
			Eventually(func(g Gomega) {
				// confirm that the namespace was created
				var ns corev1.Namespace
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: projectWithNamespace.Spec.ProjectNamespace}, &ns)).To(Succeed())
				g.Expect(ns.Name).To(Equal(projectWithNamespace.Spec.ProjectNamespace))

				By("Confirming that the namespace has the correct labels")
				// confirm that the namespace has the correct labels
				g.Expect(ns.Labels).To(SatisfyAll(
					HaveKeyWithValue("vulkan.io/project", projectWithNamespace.Name),
					HaveKeyWithValue("vulkan.io/projectID", projectWithNamespace.Spec.ProjectID),
					HaveKeyWithValue("vulkan.io/org", projectWithNamespace.Spec.OrgRef),
				))

			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
		})

		It("should create a dedicated namespace for the project if no project namespace is provided", func() {
			By("Creating a project cluster binding")
			binding := makeProjectClusterBinding(cbNamespace.Name, project.Name, cluster.Name)
			Expect(k8sClient.Create(ctx, binding)).To(Succeed())

			By("Reconciling the created resource")
			reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      binding.Name,
					Namespace: cbNamespace.Name,
				},
			})

			By("Confirming that the namespace was created")
			Eventually(func(g Gomega) {
				// confirm that the namespace was created
				var ns corev1.Namespace
				prefix := "proj-ns"
				var expectedNsName = utils.ShortName(prefix, fmt.Sprintf("%s-%s", project.Spec.OrgRef, project.Name))
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: expectedNsName}, &ns)).To(Succeed())
				g.Expect(ns.Name).To(Equal(expectedNsName))

				By("Confirming that the namespace has the correct labels")
				// confirm that the namespace has the correct labels
				g.Expect(ns.Labels).To(SatisfyAll(
					HaveKeyWithValue("vulkan.io/project", project.Name),
					HaveKeyWithValue("vulkan.io/projectID", project.Spec.ProjectID),
					HaveKeyWithValue("vulkan.io/org", project.Spec.OrgRef),
				))

			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
		})

		It("should create a resource quota for the generated namespace matching the project's resource limits", func() {
			By("Creating a project cluster binding")
			binding := makeProjectClusterBinding(cbNamespace.Name, project.Name, cluster.Name)
			Expect(k8sClient.Create(ctx, binding)).To(Succeed())

			By("Reconciling the created resource")
			reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      binding.Name,
					Namespace: cbNamespace.Name,
				},
			})

			By("Confirming that the resource quota was created")
			Eventually(func(g Gomega) {
				// confirm that the namespace was created
				var ns corev1.Namespace
				prefix := "proj-ns"
				var expectedNsName = utils.ShortName(prefix, fmt.Sprintf("%s-%s", project.Spec.OrgRef, project.Name))
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: expectedNsName}, &ns)).To(Succeed())
				g.Expect(ns.Name).To(Equal(expectedNsName))

				// confirm that the resource quota was created
				var quota corev1.ResourceQuota
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: expectedNsName,
					Name:      fmt.Sprintf("quota-%s", project.Name),
				}, &quota)).To(Succeed())
				g.Expect(quota.Name).To(Equal(fmt.Sprintf("quota-%s", project.Name)))

				By("Confirming that the resource quota has the correct limits")
				// confirm that the resource quota has the correct limits
				g.Expect(quota.Spec.Hard).To(Equal(corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse(fmt.Sprintf("%d", project.Spec.ProjectMaxCores)),
					corev1.ResourceMemory:           resource.MustParse(fmt.Sprintf("%dGi", project.Spec.ProjectMaxMemory)),
					corev1.ResourceEphemeralStorage: resource.MustParse(fmt.Sprintf("%dGi", project.Spec.ProjectMaxStorage)),
				}))
			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

		})

		It("should create a network policy for the generated namespace", func() {
			By("Creating a project cluster binding")
			binding := makeProjectClusterBinding(cbNamespace.Name, project.Name, cluster.Name)
			Expect(k8sClient.Create(ctx, binding)).To(Succeed())

			By("Reconciling the created resource")
			reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      binding.Name,
					Namespace: cbNamespace.Name,
				},
			})

			By("Confirming that the network policy was created")
			Eventually(func(g Gomega) {
				// confirm that the namespace was created
				var ns corev1.Namespace
				prefix := "proj-ns"
				var expectedNsName = utils.ShortName(prefix, fmt.Sprintf("%s-%s", project.Spec.OrgRef, project.Name))
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: expectedNsName}, &ns)).To(Succeed())
				g.Expect(ns.Name).To(Equal(expectedNsName))

				// confirm that the network policy was created
				var networkPolicy networkingv1.NetworkPolicy
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: expectedNsName,
					Name:      "vulkan-default-deny",
				}, &networkPolicy)).To(Succeed())
				g.Expect(networkPolicy.Name).To(Equal("vulkan-default-deny"))

				// confirm that the network policy has the correct rules (deny all)
				By("Confirming that the network policy has the correct rules (deny all)")
				g.Expect(networkPolicy.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
					networkingv1.PolicyTypeEgress,
				}))

			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())

		})

		It("should create role bindings for project members", func() {
			By("Creating a project cluster binding")
			binding := makeProjectClusterBinding(cbNamespace.Name, project.Name, cluster.Name)
			Expect(k8sClient.Create(ctx, binding)).To(Succeed())

			By("Reconciling the created resource")
			reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      binding.Name,
					Namespace: cbNamespace.Name,
				},
			})

			By("Confirming that the role bindings were created")
			Eventually(func(g Gomega) {
				prefix := "proj-ns"
				roleBindingNamespace := utils.ShortName(prefix, fmt.Sprintf("%s-%s", project.Spec.OrgRef, project.Name))
				// confirm that the role bindings were created
				var roleBindings rbacv1.RoleBindingList
				g.Expect(k8sClient.List(ctx, &roleBindings, client.InNamespace(roleBindingNamespace))).To(Succeed())
				g.Expect(roleBindings.Items).To(HaveLen(3))

				// confirm that the role bindings have the correct subjects and role refs
				g.Expect(roleBindings.Items[0].Subjects).To(HaveLen(1))

				g.Expect(roleBindings.Items[0].Subjects[0]).To(SatisfyAll(
					HaveField("Kind", rbacv1.UserKind),
					HaveField("Name", user1.Email),
					HaveField("APIGroup", "rbac.authorization.k8s.io"),
				))

				g.Expect(roleBindings.Items[0].RoleRef).To(SatisfyAll(
					HaveField("Kind", "ClusterRole"),
					HaveField("Name", "admin"),
				))

				g.Expect(roleBindings.Items[1].Subjects[0]).To(SatisfyAll(
					HaveField("Kind", rbacv1.UserKind),
					HaveField("Name", user2.Email),
					HaveField("APIGroup", "rbac.authorization.k8s.io"),
				))
				g.Expect(roleBindings.Items[1].RoleRef).To(SatisfyAll(
					HaveField("Kind", "ClusterRole"),
					HaveField("Name", "edit"),
				))

				g.Expect(roleBindings.Items[2].Subjects[0]).To(SatisfyAll(
					HaveField("Kind", rbacv1.UserKind),
					HaveField("Name", user3.Email),
					HaveField("APIGroup", "rbac.authorization.k8s.io"),
				))
				g.Expect(roleBindings.Items[2].RoleRef).To(SatisfyAll(
					HaveField("Kind", "ClusterRole"),
					HaveField("Name", "view"),
				))

				// confirm that the role bindings have the correct namespace
				g.Expect(roleBindings.Items[0].Namespace).To(Equal(roleBindingNamespace))
				g.Expect(roleBindings.Items[1].Namespace).To(Equal(roleBindingNamespace))
				g.Expect(roleBindings.Items[2].Namespace).To(Equal(roleBindingNamespace))

			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 200).Should(Succeed())
		})

	})

})

func buildTestProjectClusterBindingReconciler() *controllerImpl.ProjectClusterBindingReconciler {
	return &controllerImpl.ProjectClusterBindingReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		DB:     testDB,
	}
}
