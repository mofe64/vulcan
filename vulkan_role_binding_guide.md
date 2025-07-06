# Namespace-Scoped Role Binding Guide

This guide explains how to use the Vulcan platform's namespace-scoped role binding system to manage user permissions in Kubernetes clusters.

## Overview

Vulcan automatically creates Kubernetes RBAC role bindings for project members based on their project-level roles. This ensures that users can only access resources within their assigned project namespaces.

## Project Roles

Users can have three roles within a project:

| Project Role | Kubernetes Role | Permissions                                                   |
| ------------ | --------------- | ------------------------------------------------------------- |
| `admin`      | `admin`         | Full administrative access to the project namespace           |
| `maintainer` | `edit`          | Create, update, and delete resources in the project namespace |
| `viewer`     | `view`          | Read-only access to resources in the project namespace        |

## How It Works

1. **Project Creation**: When a project is created, a dedicated namespace is created for it
2. **Member Assignment**: Users are assigned roles within the project
3. **Role Binding Creation**: The system automatically creates role bindings in the project namespace
4. **Access Control**: Users can only access resources within their assigned project namespace

## Example Workflow

### 1. Create a Project

```yaml
apiVersion: platform.platform.io/v1alpha1
kind: Project
metadata:
  name: my-project
spec:
  displayName: "My Project"
  projectID: "550e8400-e29b-41d4-a716-446655440000"
  orgRef: "550e8400-e29b-41d4-a716-446655440001"
  projectMaxCores: 10
  projectMaxMemoryInGigabytes: 20
  projectMaxEphemeralStorageInGigabytes: 50
```

### 2. Add Project Members

Add users to the project with specific roles:

```sql
-- Add an admin user
INSERT INTO project_members (user_id, project_id, role)
VALUES ('user-uuid-1', '550e8400-e29b-41d4-a716-446655440000', 'admin');

-- Add a maintainer user
INSERT INTO project_members (user_id, project_id, role)
VALUES ('user-uuid-2', '550e8400-e29b-41d4-a716-446655440000', 'maintainer');

-- Add a viewer user
INSERT INTO project_members (user_id, project_id, role)
VALUES ('user-uuid-3', '550e8400-e29b-41d4-a716-446655440000', 'viewer');
```

### 3. Create Project-Cluster Binding

```yaml
apiVersion: platform.platform.io/v1alpha1
kind: ProjectClusterBinding
metadata:
  name: my-project-cluster-binding
spec:
  projectRef: my-project
  clusterRef: my-cluster
```

### 4. Automatic Role Binding Creation

The system automatically creates role bindings in the project namespace:

```yaml
# Role binding for admin user
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb-admin-admin@example.com
  namespace: proj-550e8400-e29b-41d4-a716-446655440001-my-project-550e8400-e29b-41d4-a716-446655440000
subjects:
- kind: User
  name: admin@example.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: admin
  apiGroup: rbac.authorization.k8s.io

# Role binding for maintainer user
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb-edit-maintainer@example.com
  namespace: proj-550e8400-e29b-41d4-a716-446655440001-my-project-550e8400-e29b-41d4-a716-446655440000
subjects:
- kind: User
  name: maintainer@example.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: edit
  apiGroup: rbac.authorization.k8s.io

# Role binding for viewer user
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb-view-viewer@example.com
  namespace: proj-550e8400-e29b-41d4-a716-446655440001-my-project-550e8400-e29b-41d4-a716-446655440000
subjects:
- kind: User
  name: viewer@example.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: view
  apiGroup: rbac.authorization.k8s.io
```

## Namespace Structure

Project namespaces follow this naming convention:

```
proj-{orgRef}-{projectName}-{projectID}
```

For example:

```
proj-550e8400-e29b-41d4-a716-446655440001-my-project-550e8400-e29b-41d4-a716-446655440000
```

## Role Binding Naming

Role bindings are named using this pattern:

```
rb-{k8sRole}-{userEmail}
```

Examples:

- `rb-admin-admin@example.com`
- `rb-edit-maintainer@example.com`
- `rb-view-viewer@example.com`

## Security Benefits

1. **Namespace Isolation**: Users can only access resources within their project namespace
2. **Role-Based Access**: Different permission levels based on project roles
3. **Standard Kubernetes RBAC**: Uses built-in Kubernetes ClusterRoles
4. **Audit Trail**: Clear role binding names for easy tracking
5. **Automatic Cleanup**: Role bindings are automatically removed when namespaces are deleted

## Monitoring and Troubleshooting

### Check Project Cluster Binding Status

```bash
kubectl get projectclusterbinding my-project-cluster-binding -o yaml
```

Look for status conditions:

- `Ready: True` - Role bindings created successfully
- `Error: True` - Check the message for error details

### List Role Bindings in Project Namespace

```bash
kubectl get rolebindings -n proj-{orgRef}-{projectName}-{projectID}
```

### Check User Permissions

```bash
kubectl auth can-i create pods --as=user@example.com -n proj-{orgRef}-{projectName}-{projectID}
```

### Common Issues

1. **User Email Not Found**: Ensure the user exists in the `users` table with a valid email
2. **Cluster Connection Failed**: Verify the cluster is accessible and the kubeconfig is valid
3. **Namespace Creation Failed**: Check if the namespace name is valid and doesn't conflict
4. **Role Binding Creation Failed**: Verify the user has the necessary permissions to create role bindings

## Best Practices

1. **Use Descriptive Project Names**: This helps with namespace identification
2. **Regular Role Reviews**: Periodically review project member roles
3. **Monitor Resource Usage**: Use the project quotas to prevent resource abuse
4. **Document Access Patterns**: Keep track of who needs what level of access
5. **Test Permissions**: Verify that users have the correct permissions after role changes

## API Integration

The role binding system is automatically triggered when:

- A new ProjectClusterBinding is created
- Project members are added or removed
- Project member roles are updated

No additional API calls are needed - the system handles everything automatically through the Kubernetes operator pattern.
