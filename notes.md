## notes created while building this

openapi code gen commands ->
install latest oapi-codegen `go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
Note -> go install doesn't add to path, to do this `ls "$(go env GOPATH)/bin/oapi-codegen"` to get install location and add to path
I used the one time shell command -> `export PATH="$HOME/go/bin:$PATH"` to set path for current shell session

USING oapi-codegen directtly
To generate server contract stubs -> `oapi-codegen -generate "types,gin-server" -package rest -o internal/server/server.gen.go api/openapi.yaml`
To generate client contract stubs -> `oapi-codegen -generate "types,client" -package api -o internal/api/openapi_client.gen.go api/openapi.yaml`

USING oapi-codegen go tool
install tool in projet `go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
then use add directive to go file `//go:generate go tool oapi-codegen -config specification/config.yaml specification/openapi.yaml`
then run go generate

### KUBE BUILDER

Note -> we created the kubebuilder project in another dir (operator), because we alread have code in the root,
so easist way was just to turn project into a multi-module project and have the operator in its own module
Initialise the repo as a Kubebuilder project (in operator dir) `kubebuilder init --domain platform.io --repo github.com/mofe64/vulkan/operator`

creating crds
`kubebuilder create api --group platform --version v1alpha1 --kind Org       --namespaced=false`
`kubebuilder create api --group platform --version v1alpha1 --kind Project   --namespaced=false`
`kubebuilder create api --group platform --version v1alpha1 --kind Application`
`kubebuilder create api --group platform --version v1alpha1 --kind Cluster   --namespaced=false`
`kubebuilder create api --group platform --version v1alpha1 --kind ProjectClusterBinding --namespaced=false`

Generate crds
`cd operator`
`make manifests` or `make all`

generate work file to link modules -> `go work init ./ ./operator`

## debug

### ßFixing "undefined: platformv1"

The compiler can't see the generated package. Do all three checks:

Import it in the file (often with an alias):

```
import (
    platformv1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
)
```

Make the root module aware of the operator module
Either add a replace in root go.mod:
`replace github.com/mofe64/vulkan/operator => ./operator`
Or create a Go work-space (go work init ./ ./operator) so both modules compile together.

Run go generate / make generate in operator/
This regenerates zz_generated.deepcopy.go and ensures AddToScheme exists before the root module builds.

Once the path is imported correctly and the operator code is generated, platformv1.AddToScheme will resolve and the client will be able to serialise / deserialise your CRDs.

### OPA Config

OPAURL -> eg. http://127.0.0.1:8181 (side-car), http://opa.platform.svc:8181 (cluster-wide)
The base URL of the running OPA server that will evaluate your query. This is set to wherever you decided to run OPA: usually a side-car listening on 8181, or a central OPA Service in the cluster.

PolicyPath -> eg. data/api/authz/allow
The REST endpoint under /v1/… that maps to one specific rule inside your bundle. It is built from:
data + package path + rule name → for module package api.authz, rule allow, the path is data/api/authz/allow.

Note -> for operator, run `make test` to set up envtest deps before running tests

### helm repos

cert-manager -> helm repo add jetstack https://charts.jetstack.io

### failed to get server groups: unknown" during tests

Quick-reference summary

- **Observed error**

  ```
  failed to get server groups: unknown
  ```

  (surfaces as "Failed to list nodes" when the client's discovery call fails)

- **Root cause**
  The kubeconfig stored in the Secret pointed to a **file-path CA certificate**
  (`certificate-authority: /tmp/.../ca.crt`).
  When the controller read the Secret later, that temp file no longer existed, so
  the TLS handshake with the API server failed and discovery (`GET /apis`) could
  not fetch "server groups".

- **Fix applied**
  Generate the kubeconfig with the control plane's **CA bytes embedded** as
  `certificate-authority-data` (helper `kubeconfigWithEmbeddedCA`).
  This makes the kubeconfig self-contained and immune to missing temp files.
  _(Optional for throwaway tests: set `InsecureSkipTLSVerify: true` instead.)_

Keep this three-liner handy; if you see the same discovery error in another
project, first check whether the kubeconfig is still referencing a vanished
`ca.crt` file.

# Vulcan Platform Notes

## Namespace-Scoped Role Binding System

The Vulcan platform implements a namespace-scoped role binding system that automatically creates Kubernetes RBAC role bindings for project members based on their project-level roles.

### How It Works

1. **Project Member Roles**: Users can have three roles within a project:

   - `admin`: Full administrative access to the project namespace
   - `maintainer`: Edit access to resources in the project namespace
   - `viewer`: Read-only access to resources in the project namespace

2. **Kubernetes Role Mapping**: Project roles are mapped to standard Kubernetes ClusterRoles:

   - `admin` → `admin` ClusterRole (full permissions)
   - `maintainer` → `edit` ClusterRole (create, update, delete permissions)
   - `viewer` → `view` ClusterRole (read-only permissions)

3. **Namespace Isolation**: Each project gets its own namespace where role bindings are created, ensuring that users can only access resources within their assigned project namespace.

### Implementation Details

#### Database Schema

- `project_members` table stores user-project-role relationships
- `users` table stores user information including email addresses
- The system joins these tables to get user emails for role binding creation

#### Controller Logic

The `ProjectClusterBindingReconciler` handles the creation of role bindings:

1. **Project Member Lookup**: Queries the database to get all project members with their roles and email addresses
2. **Role Mapping**: Maps project roles to Kubernetes roles
3. **Role Binding Creation**: Uses the `utils.EnsureRoleBinding` function to create namespace-scoped role bindings
4. **Error Handling**: Provides detailed error messages and status conditions for troubleshooting

#### Role Binding Structure

Each role binding follows this pattern:

- **Name**: `rb-{role}-{email}` (e.g., `rb-admin-user@example.com`)
- **Namespace**: Project-specific namespace
- **Subject**: User email address
- **Role Reference**: Standard Kubernetes ClusterRole (`admin`, `edit`, or `view`)

### Benefits

1. **Security**: Namespace isolation prevents cross-project access
2. **Simplicity**: Uses standard Kubernetes ClusterRoles, no custom roles needed
3. **Consistency**: Same permission model across all Kubernetes distributions
4. **Maintainability**: Leverages existing Kubernetes RBAC infrastructure
5. **Auditability**: Clear role binding names make it easy to track permissions

### Usage Example

When a user is added to a project with the `maintainer` role:

1. The system creates a role binding in the project namespace
2. The user gets `edit` permissions within that namespace only
3. They can create, update, and delete resources but cannot manage RBAC
4. Access is automatically restricted to the project namespace

### Troubleshooting

- Check the ProjectClusterBinding status conditions for error messages
- Verify that user emails exist in the database
- Ensure the target cluster is accessible
- Review operator logs for detailed role binding creation information

## Kubernetes Controller UID Conflict Error

### The Problem

During project deletion, the controller encountered a UID conflict error:

```
Operation cannot be fulfilled on projects.platform.platform.io: StorageError: invalid object, Code: 4,
AdditionalErrorMsg: Precondition failed: UID in precondition: d983a792-d884-4c27-8faa-060fd610eb9f, UID in object meta:
```

### Why It Occurred

The issue happened due to optimistic concurrency control in Kubernetes:

1. **Initial Get**: Controller retrieves project object with UID `d983a792-d884-4c27-8faa-060fd610eb9f`
2. **Finalizer Removal**: Controller removes finalizer and updates object via `r.Update()`
3. **UID Change**: Kubernetes assigns a new UID to the object after finalizer removal
4. **Stale Reference**: The `proj` variable still contains the old UID
5. **Failed Status Update**: Controller tries to update status using stale object metadata, causing UID mismatch

### The Fix

Removed the unnecessary status update after finalizer removal:

```go
// REMOVED - This was causing the UID conflict:
//clear error condition if it exists
apimeta.SetStatusCondition(&proj.Status.Conditions, metav1.Condition{
    Type:    platformv1alpha1.Error,
    Status:  metav1.ConditionFalse,
    Reason:  "ClusterBindingDeletionError",
    Message: "",
})
if err := r.Status().Update(ctx, &proj); err != nil {
    log.Error(err, "Failed to update project error status", "projectName", proj.Spec.DisplayName)
    return ctrl.Result{}, err
}
```

### Why This Works

- The status update was redundant since the project is being deleted anyway
- Error conditions are only set when there's an actual deletion error, in which case the controller returns early
- Eliminating the second update prevents the UID conflict entirely

### Lesson Learned

When performing multiple updates on the same object in a single reconciliation cycle, especially with finalizers and deletion timestamps, be aware that object metadata (including UID) may change between updates. Avoid unnecessary status updates during deletion phases.

### Tekton

Add the Tekton Helm repository (once per workstation/CI runner)

`helm repo add cdf https://cdfoundation.github.io/tekton-helm-chart`
`helm search repo tekton` -> to find latest version
`helm repo update`
The repo exposes the chart tekton-pipeline; at the moment the latest chart version in that repo is 1.1.0
github.com

Declare Tekton as a dependency in Chart.yaml

```
name: tekton-pipeline # chart name in the repo
  version: "1.1.0" # pin the version you saw with `helm search`
  repository: "https://cdfoundation.github.io/tekton-helm-chart" # (optional) let users switch Tekton on/off from values.yaml
```
