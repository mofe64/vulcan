# Vulkan Helm Installation Guide

This guide covers the installation, management, and removal of Vulkan using Helm charts.

## Prerequisites

Before installing Vulkan, ensure you have:

- Helm 3.x installed
- kubectl configured with access to your target cluster
- Required values files (`values.yaml` and `values.secrets.yaml`)

## Installation

### 1. Add Required Helm Repositories

Add the following repositories to your Helm configuration:

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add jetstack https://charts.jetstack.io
helm repo add tekton https://cdfoundation.github.io/tekton-helm-chart
helm repo add mycharts https://mofe64.github.io/vulcan
```

### 2. Update Helm Repositories

Fetch the latest chart versions:

```bash
helm repo update
```

### 3. Verify Chart Availability

Confirm that the Vulkan chart is available:

```bash
helm search repo mycharts
```

### 4. Install Vulkan

Install Vulkan using the Helm chart:

```bash
helm install release-name mycharts/vulkan -f values.yaml -f values.secrets.yaml
```

**Note:** Replace `release-name` with your desired release name.

#### Installing in a Specific Namespace

To install Vulkan in a specific namespace, add the `--namespace` flag:

```bash
helm install release-name mycharts/vulkan -f values.yaml -f values.secrets.yaml --namespace namespaceName
```

## Advanced Installation Options

### Using Atomic and Debug Flags

For production deployments or debugging, you can use additional flags:

- **`--atomic`**: Makes installation atomic - any failure or cancellation will rollback changes
- **`--debug`**: Provides verbose output for debugging
- **`--timeout`**: Sets a custom timeout for the installation

```bash
helm install vulkan-init mycharts/vulkan -f values.yaml -f values.secrets.yaml --timeout 10m --debug --atomic
```

## Updating Vulkan

After making updates to the Helm chart and deploying via GitHub Actions, update your local repository:

```bash
helm repo update
```

## Uninstallation

### Complete Removal Process

To completely remove Vulkan from your cluster, follow these steps in order:

#### 1. Delete Tekton Resources

```bash
# Delete Tekton namespace
kubectl delete namespace tekton-pipelines

# Delete Tekton CRDs
kubectl get crd -o name | grep tekton.dev | xargs kubectl delete
```

#### 2. Delete Cert-Manager Resources

```bash
# Delete cert-manager namespace
kubectl delete namespace cert-manager

# Delete cert-manager CRDs
kubectl get crd -o name | grep cert-manager.io | xargs kubectl delete
```

#### 3. Delete Vulkan CRDs

```bash
# Delete Vulkan CRDs (add specific CRD names as needed)
kubectl get crd -o name | grep platform.platform.io | xargs kubectl delete
```

#### 4. Uninstall Helm Release

```bash
# Uninstall the Helm release
helm uninstall release-name
```

**Note:** If you installed in a specific namespace, include the namespace flag:

```bash
helm uninstall release-name --namespace namespaceName
```

## Troubleshooting

### Common Issues

1. **Chart not found**: Ensure you've added the correct repository and run `helm repo update`
2. **Installation timeout**: Increase the timeout value or check cluster resources
3. **CRD conflicts**: Ensure all CRDs are properly removed before reinstalling

### Debug Mode

For troubleshooting installation issues, use debug mode:

```bash
helm install release-name mycharts/vulkan -f values.yaml -f values.secrets.yaml --debug
```

## Notes

- Always backup your values files before making changes
- Test installations in a non-production environment first
- Monitor the installation logs for any errors or warnings
- Ensure your cluster has sufficient resources for all components
