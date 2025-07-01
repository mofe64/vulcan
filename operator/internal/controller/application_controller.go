package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// argov1alpha1 "github.com/argoproj/argo-cd/v3.0.9/pkg/apis/application/v1alpha1"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.platform.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.platform.io,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.platform.io,resources=applications/finalizers,verbs=update

// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Reconciling Application", "namespace", req.Namespace, "name", req.Name)

	//  fetch the Application instance
	application := &platformv1alpha1.Application{}
	err := r.Get(ctx, req.NamespacedName, application)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic, see Finalizers.
			logger.Info("Application resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// define PipelineRun name and other variables
	pipelineRunName := fmt.Sprintf("%s-build-%s", application.Name, time.Now().Format("20060102150405"))
	targetImage := fmt.Sprintf("ghcr.io/mofe64/%s:%s", application.Name, application.Spec.Build.Ref)

	var buildParams []tektonv1beta1.Param
	var pipelineRef string

	buildParams = []tektonv1beta1.Param{
		{Name: "repo-url", Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: application.Spec.RepoURL}},
		{Name: "branch", Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: application.Spec.Build.Ref}},
		{Name: "image-name", Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: targetImage}},
		{Name: "gitops-repo-url", Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: "https://github.com/your-org/your-gitops-repo.git"}},
		{Name: "gitops-app-path", Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: fmt.Sprintf("apps/%s", application.Name)}},
		{Name: "app-name", Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: application.Name}},
	}

	switch application.Spec.Build.Strategy {
	case "dockerfile":
		// for dockerfile strategy, use either specified path or default to "./Dockerfile"
		dockerfilePath := "./Dockerfile" // Default path
		if application.Spec.Build.Dockerfile != "" {
			dockerfilePath = application.Spec.Build.Dockerfile
		}

		buildParams = append(buildParams, tektonv1beta1.Param{
			Name:  "dockerfile-path",
			Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: dockerfilePath},
		})

		// use dockerfile-specific pipeline
		pipelineRef = "app-build-dockerfile"

	case "buildpack":
		// for buildpack strategy, add buildpack-specific parameters
		buildParams = append(buildParams, tektonv1beta1.Param{
			Name:  "builder-image",
			Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: "paketobuildpacks/builder:base"}, // Default buildpack builder
		})

		// use buildpack-specific pipeline
		pipelineRef = "app-build-buildpack"

	default:
		return ctrl.Result{}, fmt.Errorf("invalid build strategy: %s", application.Spec.Build.Strategy)
	}

	// define the desired PipelineRun
	desiredPipelineRun := &tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: application.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "vulkan-operator",
				"app":                          application.Name,
			},
		},
		Spec: tektonv1beta1.PipelineRunSpec{
			PipelineRef: &tektonv1beta1.PipelineRef{
				Name: pipelineRef,
			},
			Params: buildParams,
			Workspaces: []tektonv1beta1.WorkspaceBinding{
				{
					Name: "shared-workspace",
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							// Define your PVC spec suitable for Tekton
							// e.g., StorageClassName and resources
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
				// You might need a workspace for Docker config or Git credentials
				{
					Name:   "docker-config",
					Secret: &corev1.SecretVolumeSource{SecretName: "docker-config-secret"}, // Your secret for GHCR access

				},
				{
					Name:   "git-credentials",
					Secret: &corev1.SecretVolumeSource{SecretName: "git-credentials-secret"}, // Your secret for GitOps repo write access

				},
			},
			// Specify a ServiceAccount for Tekton tasks if needed
			ServiceAccountName: "tekton-sa",
		},
	}

	// Set the Application as the owner of the PipelineRun.
	// This ensures that when the Application is deleted, the PipelineRun is also garbage collected.
	if err := ctrl.SetControllerReference(application, desiredPipelineRun, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for PipelineRun")
		return ctrl.Result{}, err
	}

	// Try to get the existing PipelineRun
	foundPipelineRun := &tektonv1beta1.PipelineRun{}
	err = r.Get(ctx, types.NamespacedName{Name: pipelineRunName, Namespace: application.Namespace}, foundPipelineRun)

	if err != nil && errors.IsNotFound(err) {
		// PipelineRun does not exist, create it
		logger.Info("Creating new Tekton PipelineRun", "PipelineRun.Namespace", desiredPipelineRun.Namespace, "PipelineRun.Name", desiredPipelineRun.Name)
		err = r.Create(ctx, desiredPipelineRun)
		if err != nil {
			logger.Error(err, "Failed to create new PipelineRun", "PipelineRun.Namespace", desiredPipelineRun.Namespace, "PipelineRun.Name", desiredPipelineRun.Name)
			return ctrl.Result{}, err
		}
		// PipelineRun created successfully, requeue to check its status later
		return ctrl.Result{Requeue: true}, nil // Requeue to observe status changes
	} else if err != nil {
		logger.Error(err, "Failed to get PipelineRun")
		return ctrl.Result{}, err
	}

	// --- Check if PipelineRun needs an update (e.g., if RepoURL changed in Application) ---
	// This part can get complex. For simplicity, we'll assume a new PipelineRun is always
	// triggered for major spec changes, or you might need to carefully compare desired vs found.
	// A common pattern is to delete and recreate the PipelineRun for a new build.
	// Or, more robustly, detect changes in Application.Spec.RepoURL or .Build.Ref
	// and if different, delete the old PipelineRun and create a new one.

	// For a continuous build on spec changes:
	if !reflect.DeepEqual(application.Spec.RepoURL, foundPipelineRun.Spec.Params[0].Value.StringVal) ||
		!reflect.DeepEqual(application.Spec.Build.Ref, foundPipelineRun.Spec.Params[1].Value.StringVal) {
		logger.Info("Application spec changed, deleting old PipelineRun and re-creating for new build")
		// Delete the old one
		err = r.Delete(ctx, foundPipelineRun)
		if err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete old PipelineRun for spec change")
			return ctrl.Result{}, err
		}
		// Immediately create the new one in the next reconcile loop
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("Reconciliation finished successfully!")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Register Tekton and Argo CD schemes
	if err := tektonv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Application{}).
		Owns(&tektonv1beta1.PipelineRun{}).
		Named("application").
		Complete(r)
}
