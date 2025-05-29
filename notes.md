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

generate work file to link modules -> `go work init ./ ./operator`

## debug

### ßFixing “undefined: platformv1”

The compiler can’t see the generated package. Do all three checks:

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
