Okay, here's a README document that explains your Tekton tasks, pipelines, and the custom reconciler, along with their dependencies and usage.

This README assumes you have a Helm chart structure like this:

```
your-chart/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── tekton/
│   │   ├── tekton-serviceaccount-rbac.yaml
│   │   ├── tasks/
│   │   │   ├── build-push-image.yaml (if still used, otherwise remove)
│   │   │   ├── buildpack-task.yaml
│   │   │   ├── cleanup-task.yaml
│   │   │   ├── get-git-revision.yaml
│   │   │   ├── git-clone-task.yaml
│   │   │   ├── kaniko-build-task.yaml
│   │   │   └── update-gitops-repo-task.yaml
│   │   └── pipelines/
│   │       ├── app-build-buildpack.yaml
│   │       └── app-build-dockerfile.yaml
│   └── secrets-docker-config.yaml
│   └── secrets-git-credentials.yaml
└── # your custom resource definition (CRD) and controller deployment manifests would also be here.
```

---

# Tekton-powered CI/CD for Applications

This project provides a robust CI/CD system for deploying applications, leveraging Tekton Pipelines for container image builds and a custom Kubernetes controller to automate the process based on `Application` Custom Resources. It integrates with GitOps principles by updating deployment manifests with immutable image digests.

## Table of Contents

1.  [Overview](#overview)
2.  [How It Works](#how-it-works)
3.  [Core Components](#core-components)
    - [Custom Resource Definition: `Application`](#custom-resource-definition-application)
    - [Kubernetes Controller: `ApplicationReconciler`](#kubernetes-controller-applicationreconciler)
    - [Tekton Tasks](#tekton-tasks)
    - [Tekton Pipelines](#tekton-pipelines)
4.  [Key Concepts](#key-concepts)
    - [Tekton Workspaces](#tekton-workspaces)
    - [Parameters and Results](#parameters-and-results)
    - [Immutable Image Digests for GitOps](#immutable-image-digests-for-gitops)
5.  [Dependencies](#dependencies)
6.  [Deployment](#deployment)
    - [Prerequisites](#prerequisites)
    - [Configuration (`values.yaml`)](#configuration-valuesyaml)
    - [Installation](#installation)
7.  [Usage](#usage)
    - [Creating an Application Resource](#creating-an-application-resource)
    - [Monitoring Pipeline Runs](#monitoring-pipeline-runs)
8.  [Troubleshooting](#troubleshooting)

## 1. Overview

This system automates the build and deployment of containerized applications. When an `Application` custom resource is created or updated in Kubernetes, a dedicated controller triggers a Tekton PipelineRun. This PipelineRun handles:

- Cloning source code from a Git repository.
- Building a container image using either a Dockerfile (via Kaniko) or Cloud Native Buildpacks.
- Pushing the built image to a container registry.
- Updating a GitOps repository with the new image's immutable digest, facilitating automated deployments via tools like Argo CD or Flux.

## 2. How It Works

The workflow is orchestrated as follows:

1.  **Application Resource Creation/Update:** A developer creates or updates an `Application` custom resource (CR) in Kubernetes, specifying the source code repository, desired branch/tag, build strategy (Dockerfile or Buildpack), and GitOps details.
2.  **Controller Reconciliation:** The `ApplicationReconciler` controller continuously monitors `Application` CRs.
    - When a new `Application` is found, or an existing one's `spec` changes, it initiates a reconciliation loop.
    - It determines the appropriate Tekton Pipeline (`app-build-dockerfile` or `app-build-buildpack`) based on the `build.strategy` in the `Application` CR.
    - It then constructs and creates a new `PipelineRun` resource, passing all necessary parameters (repo URL, image name, GitOps details, etc.) and binding the required workspaces (for source code, Docker config, Git credentials).
3.  **Tekton Pipeline Execution:**
    - The Tekton Pipelines controller detects the new `PipelineRun` and executes it.
    - **`git-clone-task`**: Clones the application's source code into a shared workspace.
    - **`get-git-revision`**: Extracts the exact commit SHA and branch from the cloned source.
    - **Build Task (`buildpack-task` or `kaniko-build-task`)**:
      - Uses the appropriate build tool to build the container image.
      - Pushes the image to the configured container registry.
      - **Crucially, it captures and exposes the _immutable image digest_ of the newly built image.**
    - **`update-gitops-repo-task`**:
      - Clones your GitOps repository.
      - Updates the application's deployment manifest (e.g., `deployment.yaml`) within the GitOps repository.
      - **It uses the immutable image digest (e.g., `your-image@sha256:abcdef...`) to update the image reference in the manifest**, ensuring deterministic deployments.
      - Commits and pushes these changes back to the GitOps repository.
    - **`cleanup-task`**: Runs as a `finally` task to clean up temporary files and directories in the shared workspace, regardless of pipeline success or failure.
4.  **GitOps Synchronization:** Your chosen GitOps tool (e.g., Argo CD, Flux) detects the changes in the GitOps repository and automatically synchronizes your Kubernetes cluster, deploying the new application image.

## 3. Core Components

### Custom Resource Definition: `Application`

The `Application` CRD (defined by your controller project) is the primary interface for users to define their application's build and deployment properties.

**Example `Application` CR:**

```yaml
apiVersion: platform.platform.io/v1alpha1
kind: Application
metadata:
  name: my-sample-app
  namespace: default
spec:
  repoURL: https://github.com/my-org/my-app.git
  build:
    strategy: dockerfile # or "buildpack"
    ref: main # Branch or commit SHA
    # dockerfile: ./backend/Dockerfile # Optional, for dockerfile strategy
    # contextDir: ./backend # Optional, for dockerfile strategy
    # buildArgs: "ENV=production,DEBUG=false" # Optional, for dockerfile strategy
    # envVars: "PORT=8080" # Optional, for buildpack strategy
  # Image name for the built container (base name, without tag or digest)
  # This will be combined with a tag during build, and then with a digest for GitOps.
  imageName: ghcr.io/my-org/my-app
```

### Kubernetes Controller: `ApplicationReconciler`

This is a custom controller written in Go.

- **Watches:** `Application` resources.
- **Actions:** When an `Application` is created, updated, or deleted, the reconciler is triggered. It then creates (or re-creates if spec changes) a `Tekton PipelineRun` to build and deploy the application.
- **Key Logic:**
  - Determines `build.strategy` (Dockerfile or Buildpack) and selects the appropriate Tekton Pipeline.
  - Generates a unique name for each `PipelineRun` to avoid conflicts.
  - Passes `Application` spec details as `params` to the `PipelineRun`.
  - Binds necessary `secrets` (Docker config, Git credentials) as `workspaces` to the `PipelineRun`.
  - Sets the `Application` CR as the owner of the `PipelineRun` for garbage collection.
  - Includes logic to avoid re-triggering builds if the `Application` spec has not changed and the latest build succeeded.

### Tekton Tasks

Individual reusable steps within the pipelines:

- **`git-clone-task`**:
  - **Description**: Clones a Git repository to a specified workspace.
  - **Params**: `repo-url`, `revision`.
  - **Workspaces**: `output` (where to clone), `git-credentials` (optional, for private repos).
  - **Results**: `commit` (full SHA), `url`.
- **`get-git-revision`**:
  - **Description**: Extracts the full and short commit SHA, and branch name from a cloned Git repository.
  - **Params**: `source-path`.
  - **Workspaces**: `source`.
  - **Results**: `commit`, `short-commit`, `branch`.
- **`buildpack-task`**:
  - **Description**: Builds source code into a container image using Cloud Native Buildpacks (no Dockerfile required).
  - **Params**: `image-name` (full tagged name), `builder-image`, `source-subpath`, `env-vars`.
  - **Workspaces**: `source`, `docker-config`.
  - **Results**: `image-digest` (the immutable digest).
- **`kaniko-build-task`**:
  - **Description**: Builds and pushes a container image using a Dockerfile with Kaniko (daemonless build).
  - **Params**: `image-name` (full tagged name), `dockerfile-path`, `context-dir`, `source-subpath`, `build-args`, `extra-args`.
  - **Workspaces**: `source`, `docker-config`.
  - **Results**: `image-digest` (the immutable digest).
- **`update-gitops-repo-task`**:
  - **Description**: Updates an application's manifest in a GitOps repository with the new image's immutable digest and pushes the changes.
  - **Params**: `gitops-repo-url`, `gitops-app-path`, `app-image` (the image reference with digest), `app-name`, `source-revision`.
  - **Workspaces**: `gitops-output`, `git-credentials`.
- **`cleanup-task`**:
  - **Description**: Cleans up temporary files and directories within a workspace.
  - **Workspaces**: `workspace`.

### Tekton Pipelines

Orchestrate the execution of tasks to perform a complete build and GitOps update.

- **`app-build-buildpack`**:
  - **Description**: Pipeline for building applications using Cloud Native Buildpacks.
  - **Flow**: `git-clone` -> `buildpack-build-and-push` + `get-source-revision` -> `update-gitops-repo` -> `cleanup` (finally).
  - **Params**: `repo-url`, `branch`, `image-name` (base name), `image-tag`, `builder-image`, `env-vars`, `gitops-repo-url`, `gitops-app-path`, `app-name`.
- **`app-build-dockerfile`**:
  - **Description**: Pipeline for building applications from a Dockerfile.
  - **Flow**: `git-clone` -> `kaniko-build-and-push` + `get-source-revision` -> `update-gitops-repo` -> `cleanup` (finally).
  - **Params**: `repo-url`, `branch`, `image-name` (base name), `image-tag`, `dockerfile-path`, `context-dir`, `build-args`, `gitops-repo-url`, `gitops-app-path`, `app-name`.

## 4. Key Concepts

### Tekton Workspaces

Workspaces provide a way for tasks within a pipeline to share files and state. They are typically backed by volumes (like `PersistentVolumeClaim` or `emptyDir`) or secrets/configmaps.

- **`source-workspace`**: A `VolumeClaimTemplate` based workspace used by `git-clone-task` to store the cloned source code, and subsequently used by `buildpack-task`/`kaniko-build-task` for building, and `get-git-revision` for commit info.
- **`docker-config`**: A `Secret` mounted workspace containing `~/.docker/config.json` for authenticating with container registries (used by `buildpack-task` and `kaniko-build-task`).
- **`git-credentials`**: A `Secret` mounted workspace containing `.git-credentials` for authenticating Git clone and push operations to private repositories (used by `git-clone-task` and `update-gitops-repo-task`).

### Parameters and Results

- **Parameters (`$(params.param-name)`)**: Used to pass configurable values into pipelines and tasks from their callers (e.g., the Reconciler passes `repo-url` to the Pipeline, and the Pipeline passes `image-name` to the build task).
- **Results (`$(tasks.task-name.results.result-name)`)**: Used for tasks to output information that can be consumed by subsequent tasks in the pipeline (e.g., `kaniko-build-task` outputs `image-digest` which is then used by `update-gitops-repo-task`).

### Immutable Image Digests for GitOps

This system prioritizes the use of **immutable image digests** (e.g., `my-registry.com/my-image@sha256:abcdef...`) instead of mutable tags (e.g., `my-image:latest`) when updating GitOps manifests.

- **Why?**: Mutable tags can be overwritten, meaning `my-image:latest` might refer to different image contents over time. Using an immutable digest guarantees that the exact same image bytes are deployed every time the manifest is applied, significantly improving reliability and reproducibility in GitOps environments.
- **How**: The build tasks (`buildpack-task`, `kaniko-build-task`) extract the `sha256` digest of the _actual_ pushed image. This digest is then passed to the `update-gitops-repo-task`, which writes the image reference in the `image@sha256:digest` format to your deployment manifests.

## 5. Dependencies

This Helm chart deploys the Tekton tasks, pipelines, and the necessary RBAC. It does _not_ deploy the underlying infrastructure.

You will need the following installed and configured in your Kubernetes cluster:

- **Kubernetes Cluster**: Version 1.20+ recommended.
- **Tekton Pipelines**: The core Tekton Pipelines controller must be installed in your cluster.
  - Installation instructions: [Tekton Pipelines Documentation](https://tekton.dev/docs/pipelines/install/)
- **Container Registry**: A registry where your built images will be pushed (e.g., Docker Hub, GitHub Container Registry, Quay.io, GCR, ECR).
- **GitOps Tool (Optional, but highly recommended)**: An agent that monitors your GitOps repository and applies changes to the cluster (e.g., Argo CD, Flux CD). This is what completes the "GitOps loop" by deploying the new image pushed by `update-gitops-repo-task`.
- **Vulkan Custom Controller**: The `ApplicationReconciler` and its associated Custom Resource Definition (CRD) must be deployed to the cluster. This Helm chart _assumes_ the controller is deployed elsewhere (or is part of this chart's full release).

## 6. Deployment

### Prerequisites

- Helm CLI (v3+)
- `kubectl` CLI
- Access to a Kubernetes cluster with Tekton Pipelines installed.
- `base64` command-line tool (for encoding secrets).

### Configuration (`values.yaml`)

You **must** configure the following parameters in your `values.yaml` before deploying. These are critical for Git and Docker registry authentication, and for specifying your GitOps repository.

```yaml
# values.yaml for your Helm chart

# --- Tekton Specifics ---
tekton:
  # The name of the ServiceAccount used by Tekton PipelineRuns.
  # This MUST match the `ServiceAccountName` configured in your ApplicationReconciler.
  serviceAccountName: tekton-sa

# --- Global Secrets Configuration ---
secrets:
  # Base64 encoded ~/.docker/config.json content for container registry authentication.
  # This JSON tells Kaniko/Buildpacks how to authenticate and push images.
  # Example for GitHub Container Registry (GHCR):
  # echo -n '{"auths":{"ghcr.io":{"auth":"dXNlcm5hbWU6Z2hwX1hYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhY="}}}' | base64
  # Replace 'your_base64_encoded_username_token' with your actual base64 encoded credentials (username:PAT/token)
  dockerConfigJson: "CHANGEME_YOUR_BASE64_ENCODED_DOCKER_CONFIG_JSON_HERE"

  # Git credentials for cloning and pushing to private repositories.
  # This file will be mounted by Tekton tasks.
  # Format: "https://<username>:<token>@<git-host>" (one per line)
  # Example for GitHub PAT: https://your-git-user:ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX@github.com
  # Example for GitLab PAT: https://git:oauth2-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX@gitlab.com
  gitCredentials: |
    CHANGEME_YOUR_GIT_CREDENTIALS_HERE
    # Add more lines for other Git hosts if needed

# --- GitOps Repository Configuration (used by update-gitops-repo-task) ---
gitOps:
  # The URL of your GitOps repository where application manifests are stored.
  # This repository must be accessible for cloning and pushing changes.
  repoURL: "https://github.com/your-org/your-gitops-repo.git"
  # The base path within the GitOps repository where application manifests reside.
  # For example, if your app's manifests are in 'gitops-repo/applications/my-app', this would be 'applications'.
  appPath: "applications" # Actual path will be like 'applications/my-app'
```

### Installation

1.  **Configure `values.yaml`**: Edit the `values.yaml` file with your specific secret data and GitOps repository details. **DO NOT commit sensitive secrets to source control.** Consider using a secrets management solution like Vault or Kubernetes External Secrets for production.
2.  **Install the Helm Chart**:

    ```bash
    helm install vulkan-ci ./your-chart -n your-namespace # or the namespace where your controller runs
    ```

    Replace `./your-chart` with the actual path to your Helm chart.

3.  **Verify Deployment**:
    ```bash
    kubectl get sa -n your-namespace tekton-sa
    kubectl get role -n your-namespace tekton-sa-role
    kubectl get rolebinding -n your-namespace tekton-sa-rolebinding
    kubectl get task -n your-namespace
    kubectl get pipeline -n your-namespace
    kubectl get secret -n your-namespace docker-config-secret git-credentials-secret
    ```
    Ensure all resources are created successfully.

## 7. Usage

Once the Helm chart and your custom controller are deployed, triggering a build is as simple as creating an `Application` custom resource.

### Creating an Application Resource

Create a YAML file (e.g., `my-app.yaml`) defining your application:

```yaml
# my-app.yaml
apiVersion: platform.platform.io/v1alpha1
kind: Application
metadata:
  name: my-first-application
  namespace: default # Or the namespace where you want the build to happen
spec:
  repoURL: https://github.com/tektoncd/examples.git # Replace with your application's source repo
  build:
    strategy: dockerfile # Choose "dockerfile" or "buildpack"
    ref: main # The branch or commit SHA to build
    dockerfile: ./dockerfile/hello-world/Dockerfile # For dockerfile strategy, relative path
    contextDir: ./dockerfile/hello-world # For dockerfile strategy, relative path to context
    # buildArgs: "VERSION=1.0" # Optional: comma-separated build args for dockerfile
    # envVars: "JAVA_TOOL_OPTIONS=-Xmx512m" # Optional: comma-separated env vars for buildpack
  # The base name for your image (e.g., ghcr.io/your-org/your-app)
  imageName: ghcr.io/mofe64/my-first-application
```

Apply this resource to your cluster:

```bash
kubectl apply -f my-app.yaml
```

Your `ApplicationReconciler` will detect this new resource and automatically create a `PipelineRun`.

### Monitoring Pipeline Runs

You can monitor the progress of your builds using `kubectl` or the `tkn` CLI:

1.  **Get PipelineRuns:**

    ```bash
    kubectl get pipelineruns -n default -l vulkan.io/application=my-first-application
    ```

    Or, to see all runs:

    ```bash
    tkn pr list -n default
    ```

2.  **View PipelineRun Logs:**

    ```bash
    tkn pr logs <pipeline-run-name> -f -n default
    ```

    Replace `<pipeline-run-name>` with the name of the `PipelineRun` (e.g., `my-first-application-build-20231027103045`).

3.  **Inspect Resources:**
    ```bash
    kubectl describe pipelinerun <pipeline-run-name> -n default
    kubectl describe taskrun <task-run-name-from-pipelinerun-logs> -n default
    ```

## 8. Troubleshooting

- **PipelineRun Stuck/Failed:**
  - **Check logs**: `tkn pr logs <pipelinerun-name> -n <namespace>`. This is the most common way to find errors.
  - **Check events**: `kubectl describe pipelinerun <pipelinerun-name> -n <namespace>`. Look for events that indicate PVC issues, secret mounting problems, or pod errors.
  - **RBAC**: Ensure the `tekton-sa` ServiceAccount has all necessary permissions (see `tekton-serviceaccount-rbac.yaml`). Common issues are missing permissions for `secrets`, `persistentvolumeclaims`, `pods`, or `deployments`.
  - **Workspaces**: Verify workspace names in your `ApplicationReconciler` (`source-workspace`, `docker-config`, `git-credentials`) precisely match those defined in your Tekton Pipelines.
  - **Secrets**:
    - Confirm `docker-config-secret` and `git-credentials-secret` exist in the correct namespace (`kubectl get secret -n <namespace>`).
    - Verify their content is correctly base64 encoded and formatted. For `.dockerconfigjson`, it must be valid JSON before base64 encoding. For `.git-credentials`, ensure correct `https://user:token@host` format.
- **Git Authentication Issues**:
  - Error messages like "Authentication failed", "permission denied".
  - Double-check your `git-credentials-secret` content. The PAT/token must have the correct scopes (e.g., `repo` for GitHub, `api` or `write_repository` for GitLab).
  - Ensure the Git host in the credential URL matches the `repo-url` in your `Application` spec.
- **Container Registry Authentication Issues**:
  - Error messages like "denied: access forbidden", "authentication required".
  - Double-check your `docker-config-secret` content.
  - Ensure the credentials have push/pull access to the specified registry and repository.
- **`update-gitops-repo-task` Failures**:
  - If the task fails to clone the GitOps repo, it's likely a `git-credentials` issue.
  - If it fails to push, ensure the configured Git user/token has write access to the GitOps repository.
  - If `yq` or `sed` errors, check the YAML structure it's trying to modify or ensure `yq` is actually available in the `bitnami/kubectl` image.
- **Reconciler not creating `PipelineRun`**:
  - Check your controller's logs (`kubectl logs <your-controller-pod-name> -n <controller-namespace>`).
  - Verify the `Application` CRD is installed correctly.
  - Ensure the reconciler has RBAC permissions to `create` and `delete` `pipelineruns`.
  - Check the reconciliation logic (e.g., if a previous successful run exists, it might not re-trigger).
