# Vulkan Platform Helm Chart

A self-hosted Platform as a Service (PaaS) control plane built on Kubernetes.

## Overview

Vulkan provides a complete GitOps-driven platform for managing applications, organizations, projects, and clusters. It integrates with popular cloud-native tools to provide a seamless developer experience.

## Architecture

- **Vulkan API**: Backend API server for platform management
- **Vulkan UI**: Web interface for the platform
- **Dex**: OIDC identity provider with GitHub integration
- **Tekton**: CI/CD pipelines for building and deploying applications
- **Argo CD**: GitOps deployment engine
- **NATS**: Message broker for internal communication
- **Crossplane**: Cloud resource management
- **NGINX Ingress**: Traffic routing and SSL termination

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- cert-manager (for SSL certificates)
- PostgreSQL database (for Vulkan API)

## Quick Start

### Development Deployment

```bash
# Install with development settings
helm install vulkan ./charts/vulkan -f values.yaml -f values-dev.yaml
```

### Production Deployment

```bash
# 1. Create secrets file (DO NOT COMMIT)
cp values-secrets.yaml.example values-secrets.yaml
# Edit values-secrets.yaml with your actual secrets

# 2. Install with production settings
helm install vulkan ./charts/vulkan -f values.yaml -f values-prod.yaml -f values-secrets.yaml
```

## Configuration

### Environment-Specific Values

- `values.yaml` - Base configuration
- `values-dev.yaml` - Development settings (insecure, mock secrets)
- `values-prod.yaml` - Production settings (secure, higher resources)
- `values-secrets.yaml` - Secrets (create from template, DO NOT COMMIT)

### Key Configuration Options

#### Global Settings

```yaml
global:
  domain: vulkan.strawhatengineer.com
  imageRegistry: ghcr.io/mofe64/vulkan
  email: your-email@example.com
```

#### API Configuration

```yaml
api:
  replicas: 3
  env:
    DATABASE_URL: "postgres://user:pass@host:port/db"
    OPA_URL: "http://127.0.0.1:8181"
    NATS_URL: "nats://nats:4222"
```

#### UI Configuration

```yaml
ui:
  replicas: 2
  apiBase: "https://api.vulkan.strawhatengineer.com"
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
```

## Security

### Required Secrets

1. **Database Password**: PostgreSQL connection string
2. **Dex Client Secret**: OIDC client secret for API
3. **GitHub OAuth**: Client ID and secret for GitHub login
4. **Argo CD Admin Password**: bcrypt hash for admin access

### Generate Argo CD Admin Password

```bash
# Install htpasswd utility
# Ubuntu/Debian: apt-get install apache2-utils
# macOS: brew install httpd

# Generate bcrypt hash
htpasswd -nbBC 12 "" your-password-here | cut -d ":" -f 2
```

## Custom Resource Definitions

Vulkan defines several CRDs for platform management:

- `Application` - Git-based application deployments
- `Organization` - Multi-tenant organization management
- `Project` - Project-level resource grouping
- `Cluster` - Kubernetes cluster management

## Troubleshooting

### Common Issues

1. **Database Connection**: Ensure PostgreSQL is accessible and credentials are correct
2. **OIDC Configuration**: Verify Dex configuration and GitHub OAuth settings
3. **SSL Certificates**: Check cert-manager is properly configured
4. **Resource Limits**: Monitor resource usage and adjust limits as needed

### Logs

```bash
# Check API logs
kubectl logs -l app.kubernetes.io/name=vulkan-api

# Check UI logs
kubectl logs -l app.kubernetes.io/name=vulkan-ui-deployment

# Check Dex logs
kubectl logs -l app=dex
```

## Development

### Local Development

1. Use `values-dev.yaml` for insecure development
2. Mock secrets are provided for local testing
3. `--insecure` flag is enabled for Argo CD

### Building Images

```bash
# Build API image
docker build -t ghcr.io/mofe64/vulkan-api:latest ./api

# Build UI image
docker build -t ghcr.io/mofe64/vulkan-ui:latest ./ui
```

## License

[Add your license here]
