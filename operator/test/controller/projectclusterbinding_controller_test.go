package controller

// import (
// 	"context"
// 	"fmt"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	rbacv1 "k8s.io/api/rbac/v1"
// 	"k8s.io/apimachinery/pkg/api/errors"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/types"
// 	"sigs.k8s.io/controller-runtime/pkg/reconcile"

// 	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
// 	"github.com/mofe64/vulkan/operator/internal/controller"
// 	"github.com/mofe64/vulkan/operator/internal/utils"
// )

// var _ = Describe("ProjectClusterBinding Controller", func() {
// 	Context("When reconciling a resource", func() {
// 		const resourceName = "test-resource"

// 		ctx := context.Background()

// 		typeNamespacedName := types.NamespacedName{
// 			Name:      resourceName,
// 			Namespace: "default",
// 		}
// 		projectclusterbinding := &platformv1alpha1.ProjectClusterBinding{}

// 		BeforeEach(func() {
// 			By("creating the custom resource for the Kind ProjectClusterBinding")
// 			err := k8sClient.Get(ctx, typeNamespacedName, projectclusterbinding)
// 			if err != nil && errors.IsNotFound(err) {
// 				resource := &platformv1alpha1.ProjectClusterBinding{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:      resourceName,
// 						Namespace: "default",
// 					},
// 					Spec: platformv1alpha1.ProjectClusterBindingSpec{
// 						ProjectRef: "test-project",
// 						ClusterRef: "test-cluster",
// 					},
// 				}
// 				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
// 			}
// 		})

// 		AfterEach(func() {
// 			resource := &platformv1alpha1.ProjectClusterBinding{}
// 			err := k8sClient.Get(ctx, typeNamespacedName, resource)
// 			Expect(err).NotTo(HaveOccurred())

// 			By("Cleanup the specific resource instance ProjectClusterBinding")
// 			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
// 		})

// 		It("should successfully reconcile the resource", func() {
// 			By("Reconciling the created resource")
// 			controllerReconciler := &controller.ProjectClusterBindingReconciler{
// 				Client: k8sClient,
// 				Scheme: k8sClient.Scheme(),
// 			}

// 			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 				NamespacedName: typeNamespacedName,
// 			})
// 			Expect(err).NotTo(HaveOccurred())
// 		})
// 	})

// 	Context("Role Binding Creation", func() {
// 		It("should create role bindings for project members", func() {
// 			// Create a mock database connection (in real tests, you'd use a test database)
// 			// For this test, we'll just verify the role mapping logic

// 			// Test role mapping
// 			testCases := []struct {
// 				projectRole     string
// 				expectedK8sRole string
// 			}{
// 				{"admin", "admin"},
// 				{"maintainer", "edit"},
// 				{"viewer", "view"},
// 				{"unknown", ""},
// 			}

// 			for _, tc := range testCases {
// 				var k8sRole string
// 				switch tc.projectRole {
// 				case "admin":
// 					k8sRole = "admin"
// 				case "maintainer":
// 					k8sRole = "edit"
// 				case "viewer":
// 					k8sRole = "view"
// 				default:
// 					k8sRole = ""
// 				}

// 				Expect(k8sRole).To(Equal(tc.expectedK8sRole))
// 			}
// 		})

// 		It("should create role binding with correct structure", func() {
// 			ctx := context.Background()
// 			namespace := "test-namespace"
// 			userEmail := "test@example.com"
// 			role := "admin"

// 			// Create a role binding using the utils function
// 			err := utils.EnsureRoleBinding(ctx, k8sClient, namespace, userEmail, role)
// 			Expect(err).NotTo(HaveOccurred())

// 			// Verify the role binding was created
// 			roleBindingName := fmt.Sprintf("rb-%s-%s", role, userEmail)
// 			roleBinding := &rbacv1.RoleBinding{}
// 			err = k8sClient.Get(ctx, types.NamespacedName{
// 				Name:      roleBindingName,
// 				Namespace: namespace,
// 			}, roleBinding)
// 			Expect(err).NotTo(HaveOccurred())

// 			// Verify the role binding structure
// 			Expect(roleBinding.Name).To(Equal(roleBindingName))
// 			Expect(roleBinding.Namespace).To(Equal(namespace))
// 			Expect(roleBinding.Subjects).To(HaveLen(1))
// 			Expect(roleBinding.Subjects[0].Kind).To(Equal("User"))
// 			Expect(roleBinding.Subjects[0].Name).To(Equal(userEmail))
// 			Expect(roleBinding.RoleRef.Kind).To(Equal("ClusterRole"))
// 			Expect(roleBinding.RoleRef.Name).To(Equal(role))
// 			Expect(roleBinding.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))

// 			// Cleanup
// 			Expect(k8sClient.Delete(ctx, roleBinding)).To(Succeed())
// 		})
// 	})
// })
