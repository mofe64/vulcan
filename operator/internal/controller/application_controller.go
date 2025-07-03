package controller

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// argov1alpha1 "github.com/argoproj/argo-cd/v3.0.9/pkg/apis/application/v1alpha1"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
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

// add permissions for PVC creation by controller
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

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
	// pipelineRunName := fmt.Sprintf("%s-build-%s", application.Name, time.Now().Format("20060102150405"))
	// Base image name (without tag)
	baseImageName := fmt.Sprintf("ghcr.io/mofe64/%s", application.Name)

	// Use the application's build ref as the initial image tag
	// This tag will be used *during* the build step of the pipeline.
	// The GitOps update will use the digest for immutability.
	imageTag := application.Spec.Build.Ref

	var buildParams []tektonv1.Param
	var pipelineRef string

	buildParams = []tektonv1.Param{
		{Name: "repo-url", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: application.Spec.RepoURL}},
		{Name: "branch", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: application.Spec.Build.Ref}},
		{Name: "image-name", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: baseImageName}},
		{Name: "image-tag", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: imageTag}},
		{Name: "gitops-repo-url", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "https://github.com/mofe64/vulcan-gitops-repo.git"}},
		{Name: "gitops-app-path", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: fmt.Sprintf("apps/%s", application.Name)}},
		{Name: "app-name", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: application.Name}},
	}

	switch application.Spec.Build.Strategy {
	case "dockerfile":
		// for dockerfile strategy, use either specified path or default to "./Dockerfile"
		dockerfilePath := "./Dockerfile" // Default path
		if application.Spec.Build.Dockerfile != "" {
			dockerfilePath = application.Spec.Build.Dockerfile
		}

		buildParams = append(buildParams, tektonv1.Param{
			Name:  "dockerfile-path",
			Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: dockerfilePath},
		})
		// Add context-dir and build-args if your custom resource supports them
		// For example:
		// if application.Spec.Build.ContextDir != "" {
		// 	buildParams = append(buildParams, tektonv1beta1.Param{
		// 		Name:  "context-dir",
		// 		Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: application.Spec.Build.ContextDir},
		// 	})
		// }
		// if application.Spec.Build.BuildArgs != "" {
		// 	buildParams = append(buildParams, tektonv1beta1.Param{
		// 		Name:  "build-args",
		// 		Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: application.Spec.Build.BuildArgs},
		// 	})
		// }

		// use dockerfile-specific pipeline
		pipelineRef = "app-build-dockerfile"

	case "buildpack":
		// for buildpack strategy, add buildpack-specific parameters
		buildParams = append(buildParams, tektonv1.Param{
			Name:  "builder-image",
			Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "paketobuildpacks/builder:base"}, // Default buildpack builder
		})
		// Add env-vars
		// For example:
		// if application.Spec.Build.EnvVars != "" {
		// 	buildParams = append(buildParams, tektonv1beta1.Param{
		// 		Name:  "env-vars",
		// 		Value: tektonv1beta1.ParamValue{Type: tektonv1beta1.ParamTypeString, StringVal: application.Spec.Build.EnvVars},
		// 	})
		// }

		// use buildpack-specific pipeline
		pipelineRef = "app-build-buildpack"

	default:
		return ctrl.Result{}, fmt.Errorf("invalid build strategy: %s", application.Spec.Build.Strategy)
	}

	// define the desired PipelineRun
	desiredPipelineRun := &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-build-", application.Name), // Better for multiple runs
			Namespace:    application.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "vulkan-operator",
				"app":                          application.Name,
			},
		},
		Spec: tektonv1.PipelineRunSpec{
			PipelineRef: &tektonv1.PipelineRef{
				Name: pipelineRef,
			},
			Params: buildParams,
			Workspaces: []tektonv1.WorkspaceBinding{
				{
					Name: "source-workspace",
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
					Secret: &corev1.SecretVolumeSource{SecretName: "vulkan-docker-config-secret"}, // Your secret for GHCR access

				},
				{
					Name:   "git-credentials",
					Secret: &corev1.SecretVolumeSource{SecretName: "vulkan-git-credentials-secret"}, // Your secret for GitOps repo write access

				},
			},
			TaskRunTemplate: tektonv1.PipelineTaskRunTemplate{
				ServiceAccountName: "tekton-sa",
			},
		},
	}

	// Set the Application as the owner of the PipelineRun.
	// This ensures that when the Application is deleted, the PipelineRun is also garbage collected.
	if err := ctrl.SetControllerReference(application, desiredPipelineRun, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for PipelineRun")
		return ctrl.Result{}, err
	}

	// Check if a build is already in progress or if spec has changed.
	// For continuous builds on spec changes, it's often simpler to just
	// *always* create a new PipelineRun if the spec is different from the last *successful* one.
	// This simplified logic checks for an *existing* PipelineRun linked to this Application
	// that matches the current desired parameters.

	// A more robust reconciliation for builds typically involves:
	// 1. Finding the latest PipelineRun for this Application.
	// 2. Checking its status (Succeeded, Failed, Running).
	// 3. If Succeeded AND Application.Spec matches the PipelineRun's params, do nothing.
	// 4. If Failed OR Application.Spec has changed, create a NEW PipelineRun.
	// 5. If Running, do nothing (wait for it to complete).

	// For simplicity, let's implement a pattern that triggers a new build if:
	// - No existing PipelineRun for this Application
	// - OR the latest PipelineRun for this Application has different build-related parameters
	// - OR the latest PipelineRun for this Application has failed or cancelled

	// List PipelineRuns owned by this Application
	existingRuns := &tektonv1.PipelineRunList{}
	listOpts := []client.ListOption{
		client.InNamespace(application.Namespace),
		client.MatchingLabels(map[string]string{"vulkan.io/application": application.Name}),
	}
	if err := r.List(ctx, existingRuns, listOpts...); err != nil {
		logger.Error(err, "Failed to list PipelineRuns for Application")
		return ctrl.Result{}, err
	}

	var latestRun *tektonv1.PipelineRun
	for i := range existingRuns.Items {
		run := &existingRuns.Items[i]
		if latestRun == nil || run.CreationTimestamp.After(latestRun.CreationTimestamp.Time) {
			latestRun = run
		}
	}

	shouldCreateNewRun := true
	if latestRun != nil {
		// Check if the latest run is still running or succeeded and matches current spec
		isFinished := latestRun.Status.CompletionTime != nil
		isSucceeded := false
		successCondition := latestRun.Status.GetCondition(apis.ConditionSucceeded)
		if isFinished && successCondition != nil && successCondition.IsTrue() {
			isSucceeded = true
		}

		// Compare current desired parameters with the latest PipelineRun's parameters
		// This requires iterating and comparing all relevant parameters
		currentParamsMap := make(map[string]string)
		for _, p := range buildParams {
			currentParamsMap[p.Name] = p.Value.StringVal
		}

		latestRunParamsMap := make(map[string]string)
		for _, p := range latestRun.Spec.Params {
			latestRunParamsMap[p.Name] = p.Value.StringVal
		}

		paramsMatch := reflect.DeepEqual(currentParamsMap, latestRunParamsMap)

		if !isFinished || (isSucceeded && paramsMatch) {
			// If a run is still active, or if the latest successful run matches parameters,
			// we don't need to create a new one.
			logger.Info("Existing PipelineRun is active, succeeded and matches, or still running. Skipping new run creation.",
				"PipelineRun.Name", latestRun.Name, "Succeeded", isSucceeded, "Finished", isFinished, "ParamsMatch", paramsMatch)
			shouldCreateNewRun = false
		}
	}

	if shouldCreateNewRun {
		logger.Info("Creating new Tekton PipelineRun", "PipelineRun.GenerateName", desiredPipelineRun.GenerateName)
		err = r.Create(ctx, desiredPipelineRun)
		if err != nil {
			logger.Error(err, "Failed to create new PipelineRun", "PipelineRun.GenerateName", desiredPipelineRun.GenerateName)
			return ctrl.Result{}, err
		}
		logger.Info("New PipelineRun created successfully", "PipelineRun.Name", desiredPipelineRun.Name)
		// Requeue immediately to observe the newly created PipelineRun
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("Reconciliation finished successfully! No new PipelineRun needed.")
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
		Owns(&tektonv1.PipelineRun{}).
		Named("application").
		Complete(r)
}
