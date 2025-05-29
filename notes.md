## notes created while building this

openapi code gen commands ->
install latest oapi-codegen `go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
Note -> go install doesn't add to path, to do this `ls "$(go env GOPATH)/bin/oapi-codegen"` to get install location and add to path
I used the one time shell command -> `export PATH="$HOME/go/bin:$PATH"` to set path for current shell session

USING oapi-codegen directtly
To generate server contract stubs -> `oapi-codegen -generate "types,gin-server" -package rest -o pkg/server/server.gen.go api/openapi.yaml`
To generate client contract stubs -> `oapi-codegen -generate "types,client" -package api -o pkg/api/openapi_client.gen.go api/openapi.yaml`

USING oapi-codegen go tool
install tool in projet `go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
then use add directive to go file `//go:generate go tool oapi-codegen -config specification/config.yaml specification/openapi.yaml`
then run go generate
