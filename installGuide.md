###install cert manager
helm install cert-manager jetstack/cert-manager \
 --namespace cert-manager --create-namespace \
 --version v1.14.4 \
 --set installCRDs=true

###install tekton
helm install tekton-pipeline cdf/tekton-pipeline \
 --namespace tekton-pipelines --create-namespace \
 --version 1.1.0

helm install vulkan-init my-charts/vulkan -f values.yaml -f values.secrets.yaml -n vulkan-namespace # Ensure you specify your app's namespace
