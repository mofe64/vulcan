# Helm Chart Publishing & Management Guide

## Overview

This guide covers the complete workflow for publishing, installing, and managing Helm charts using GitHub Actions and GitHub Pages as a Helm repository.

## Table of Contents

1. [Publishing Workflow](#publishing-workflow)
2. [Installation Guide](#installation-guide)
3. [Upgrading Charts](#upgrading-charts)
4. [Best Practices](#best-practices)
5. [Troubleshooting](#troubleshooting)
6. [Advanced Topics](#advanced-topics)

---

## Publishing Workflow

### How the GitHub Actions Workflow Works

Our GitHub Actions workflow (`publish-helm-charts.yaml`) automates the entire publishing process:

#### Trigger Conditions

- **Automatic**: Pushes to `main` branch that modify:
  - `charts/vulkan-charts/**` (chart files)
  - `.github/workflows/publish-helm-charts.yaml` (workflow file)
- **Manual**: Via GitHub UI using `workflow_dispatch`

#### Step-by-Step Process

1. **Checkout**: Fetches the complete repository history
2. **Setup**: Installs required tools (yq, Helm)
3. **Package**:
   - Updates chart dependencies
   - Extracts chart name and version from `Chart.yaml`
   - Creates a `.tgz` package
4. **Prepare Repository**:
   - Creates repository structure for GitHub Pages
   - Generates `index.yaml` with proper URLs
5. **Publish**: Deploys to `gh-pages` branch using GitHub Pages

#### Generated Structure

```
gh-pages branch/
├── vulkan-0.1.0.tgz    # Packaged chart
└── index.yaml          # Repository index
```

### Chart Versioning

**Critical**: Always increment the version in `Chart.yaml` before making changes:

```yaml
# charts/vulkan-charts/Chart.yaml
apiVersion: v2
name: vulkan
description: Self-hostable PaaS control-plane
type: application
version: 0.1.1 # <- Increment this!
appVersion: 0.1.0
```

### Publishing Process

1. Make changes to your chart
2. Update version in `Chart.yaml`
3. Commit and push to `main`:
   ```bash
   git add .
   git commit -m "Update chart: description of changes"
   git push origin main
   ```
4. GitHub Actions automatically publishes the new version

---

## Installation Guide

### First-Time Setup

```bash
# 1. Add the Helm repository
helm repo add my-charts https://mofe64.github.io/vulcan

# 2. Update repository index
helm repo update

# 3. Verify chart availability
helm search repo my-charts
```

### Basic Installation

```bash
# Install with default values
helm install my-vulkan my-charts/vulkan

# Install in specific namespace
helm install my-vulkan my-charts/vulkan \
  --namespace vulkan \
  --create-namespace
```

### Installation with Custom Values

```bash
# Using values file
helm install my-vulkan my-charts/vulkan -f custom-values.yaml

# Using inline values
helm install my-vulkan my-charts/vulkan \
  --set key1=value1 \
  --set key2=value2

# Using multiple values files
helm install my-vulkan my-charts/vulkan \
  -f base-values.yaml \
  -f env-specific-values.yaml
```

### Pre-Installation Checks

```bash
# View available chart versions
helm search repo my-charts/vulkan --versions

# Check chart values
helm show values my-charts/vulkan

# Dry run installation
helm install my-vulkan my-charts/vulkan \
  --dry-run \
  --debug
```

---

## Upgrading Charts

### Standard Upgrade Process

```bash
# 1. Update repository cache
helm repo update

# 2. Check for new versions
helm search repo my-charts/vulkan --versions

# 3. Upgrade to latest version
helm upgrade my-vulkan my-charts/vulkan

# 4. Verify upgrade
helm status my-vulkan
```

### Upgrade Options

```bash
# Upgrade to specific version
helm upgrade my-vulkan my-charts/vulkan --version 0.1.2

# Upgrade with custom values
helm upgrade my-vulkan my-charts/vulkan -f new-values.yaml

# Atomic upgrade (rollback on failure)
helm upgrade my-vulkan my-charts/vulkan --atomic

# Upgrade with timeout
helm upgrade my-vulkan my-charts/vulkan --timeout 10m

# Wait for all resources to be ready
helm upgrade my-vulkan my-charts/vulkan --wait

# Preview changes before applying
helm upgrade my-vulkan my-charts/vulkan --dry-run
```

### Upgrade Safety

```bash
# Reset values to chart defaults
helm upgrade my-vulkan my-charts/vulkan --reset-values

# Reuse existing values and merge with new ones
helm upgrade my-vulkan my-charts/vulkan --reuse-values

# Force resource update
helm upgrade my-vulkan my-charts/vulkan --force
```

---

## Best Practices

### Chart Development

1. **Version Management**:

   - Use semantic versioning (MAJOR.MINOR.PATCH)
   - Increment version for every change
   - Update `appVersion` when application changes

2. **Testing**:

   - Always test with `--dry-run` first
   - Use `helm lint` to validate chart syntax
   - Test in development environment before production

3. **Documentation**:
   - Maintain clear `README.md` in chart directory
   - Document all configurable values
   - Include examples of common configurations

### Installation Best Practices

1. **Namespacing**:

   ```bash
   # Always use dedicated namespaces
   helm install my-vulkan my-charts/vulkan \
     --namespace vulkan \
     --create-namespace
   ```

2. **Values Management**:

   ```bash
   # Use environment-specific values files
   helm install my-vulkan my-charts/vulkan \
     -f base-values.yaml \
     -f production-values.yaml
   ```

3. **Resource Management**:
   - Monitor resource usage after installation
   - Set appropriate resource limits and requests
   - Consider node affinity and tolerations

### Operational Best Practices

1. **Release Management**:

   ```bash
   # Use meaningful release names
   helm install vulkan-prod my-charts/vulkan  # Not "release-1"

   # Track release history
   helm history vulkan-prod
   ```

2. **Monitoring**:

   ```bash
   # Regular health checks
   helm status vulkan-prod
   helm list --all-namespaces
   ```

3. **Backup Strategy**:
   - Keep copies of your values files
   - Document custom configurations
   - Test rollback procedures

---

## Troubleshooting

### Common Issues

#### 1. Repository Not Found (404 Error)

```bash
# Error: failed to fetch https://mofe64.github.io/vulcan/charts/index.yaml : 404 Not Found
```

**Solution**: Check GitHub Pages deployment and URL structure

#### 2. Chart Version Not Available

```bash
# Error: chart "vulkan" matching 0.1.2 not found
```

**Solutions**:

```bash
# Update repository cache
helm repo update

# Check available versions
helm search repo my-charts/vulkan --versions

# Verify GitHub Actions completed successfully
```

#### 3. Upgrade Failures

```bash
# Error: UPGRADE FAILED: another operation is in progress
```

**Solutions**:

```bash
# Check release status
helm status my-vulkan

# Force upgrade if stuck
helm upgrade my-vulkan my-charts/vulkan --force

# Rollback if needed
helm rollback my-vulkan
```

#### 4. Dependency Issues

```bash
# Error: found in Chart.yaml, but missing in charts/ directory
```

**Solution**: Dependencies are automatically handled by the GitHub Actions workflow

### Debugging Commands

```bash
# Check release history
helm history my-vulkan

# Get release manifest
helm get manifest my-vulkan

# Get release values
helm get values my-vulkan

# Debug with verbose output
helm install my-vulkan my-charts/vulkan --debug --dry-run
```

---

## Advanced Topics

### Rollback Strategies

```bash
# Rollback to previous version
helm rollback my-vulkan

# Rollback to specific revision
helm rollback my-vulkan 2

# Rollback with dry-run
helm rollback my-vulkan --dry-run
```

### Multi-Environment Management

```bash
# Development
helm install vulkan-dev my-charts/vulkan \
  -f values-dev.yaml \
  --namespace vulkan-dev

# Staging
helm install vulkan-staging my-charts/vulkan \
  -f values-staging.yaml \
  --namespace vulkan-staging

# Production
helm install vulkan-prod my-charts/vulkan \
  -f values-prod.yaml \
  --namespace vulkan-prod
```

### Chart Dependencies

Your chart includes several dependencies:

- **nats**: Message broker
- **crossplane**: Kubernetes control plane
- **argocd**: GitOps continuous delivery
- **dex**: Identity service (conditional)
- **nginx**: Ingress controller
- **tekton**: CI/CD pipeline
- **cert-manager**: Certificate management (conditional)

These are automatically managed by Helm during installation.

### Custom Repository Management

```bash
# Add multiple repositories
helm repo add my-charts https://mofe64.github.io/vulcan
helm repo add bitnami https://charts.bitnami.com/bitnami

# List repositories
helm repo list

# Remove repository
helm repo remove my-charts

# Update all repositories
helm repo update
```

### Monitoring and Maintenance

```bash
# Regular maintenance commands
helm repo update                    # Weekly
helm list --all-namespaces        # Monitor releases
helm history my-vulkan             # Track changes

# Cleanup unused releases
helm uninstall old-release
```

---

## Quick Reference

### Essential Commands

```bash
# Repository management
helm repo add my-charts https://mofe64.github.io/vulcan
helm repo update
helm search repo my-charts

# Installation
helm install my-vulkan my-charts/vulkan
helm install my-vulkan my-charts/vulkan -f values.yaml --namespace vulkan

# Upgrade
helm upgrade my-vulkan my-charts/vulkan
helm upgrade my-vulkan my-charts/vulkan --version 0.1.2

# Management
helm status my-vulkan
helm history my-vulkan
helm rollback my-vulkan
helm uninstall my-vulkan
```

### Important URLs

- **Chart Repository**: https://mofe64.github.io/vulcan
- **GitHub Repository**: https://github.com/mofe64/vulcan
- **GitHub Actions**: https://github.com/mofe64/vulcan/actions

This comprehensive guide should serve as your go-to reference for managing your Helm chart publishing and deployment workflow.
