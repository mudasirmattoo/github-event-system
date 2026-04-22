# Kubernetes Secrets Management Guide

## Overview
This guide demonstrates how to use Kubernetes Secrets to securely manage environment variables for the GitHub Event System.

## Why Use Secrets?

### Security Benefits
- **Encrypted Storage**: Secrets are stored encrypted in etcd
- **Access Control**: Fine-grained RBAC permissions per secret
- **No Exposure**: Not visible in manifests or pod specs
- **Audit Trail**: Secret access is logged and auditable
- **Isolation**: Separate from application code and configs

### Operational Benefits
- **Environment-Specific**: Different secrets for dev/staging/prod
- **Easy Rotation**: Update secrets without restarting pods
- **Version Control Safe**: No sensitive data in Git history
- **Multi-Source**: Support files, literals, or external secret stores

## Implementation

### 1. Create Secrets

```bash
kubectl create secret generic github-event-secrets \
  --from-literal=DB_HOST=postgres \
  --from-literal=DB_PORT=5432 \
  --from-literal=DB_NAME=github_events \
  --from-literal=DB_USER=postgres \
  --from-literal=DB_PASSWORD=postgres \
  --from-literal=REDIS_HOST=redis \
  --from-literal=REDIS_PORT=6379
```

### 2. Reference Secrets in Deployments

#### API Deployment (k8s/api.yaml)
```yaml
env:
  - name: REDIS_HOST
    valueFrom:
      secretKeyRef:
        name: github-event-secrets
        key: REDIS_HOST
  - name: DB_HOST
    valueFrom:
      secretKeyRef:
        name: github-event-secrets
        key: DB_HOST
  # ... other environment variables
```

#### Worker Deployment (k8s/worker.yaml)
```yaml
env:
  - name: REDIS_HOST
    valueFrom:
      secretKeyRef:
        name: github-event-secrets
        key: REDIS_HOST
  - name: DB_HOST
    valueFrom:
      secretKeyRef:
        name: github-event-secrets
        key: DB_HOST
  # ... other environment variables
```

## Secret Management Operations

### View Existing Secrets
```bash
# List all secrets
kubectl get secrets

# Describe specific secret
kubectl describe secret github-event-secrets

# Get secret in YAML format
kubectl get secret github-event-secrets -o yaml
```

### Update Secrets
```bash
# Add new key to existing secret
kubectl patch secret github-event-secrets --patch '{"data":{"NEW_KEY":"bmV3X3ZhbHVl"}}'

# Update existing key
kubectl patch secret github-event-secrets --patch '{"data":{"DB_PASSWORD":"bmV3UGFzc3dvcmQ=="}}'

# Delete and recreate
kubectl delete secret github-event-secrets
kubectl create secret generic github-event-secrets --from-literal=...
```

### Delete Secrets
```bash
kubectl delete secret github-event-secrets
```

## Advanced Usage

### Environment-Specific Secrets
```bash
# Development
kubectl create secret github-event-secrets-dev --from-literal=DB_HOST=postgres-dev...

# Staging
kubectl create secret github-event-secrets-staging --from-literal=DB_HOST=postgres-staging...

# Production
kubectl create secret github-event-secrets-prod --from-literal=DB_HOST=postgres-prod...
```

### Using Different Secret Sources

#### From Files
```bash
kubectl create secret generic github-event-secrets \
  --from-file=./config/db-credentials.txt \
  --from-file=./config/redis-credentials.txt
```

#### From Environment Variables
```bash
# Create from current environment
kubectl create secret generic github-event-secrets \
  --from-env-file=.env.production
```

#### From External Secret Store
```bash
# Using external secret manager
kubectl create secret generic github-event-secrets \
  --from-literal=DB_HOST=$(aws secretsmanager get-secret-value...)
```

## Best Practices

### 1. Naming Conventions
- Use descriptive names: `app-name-secrets` or `app-name-config`
- Include environment: `app-secrets-dev`, `app-secrets-prod`
- Use consistent naming across environments

### 2. Access Control
```yaml
# Create secret with restricted access
apiVersion: v1
kind: Secret
metadata:
  name: github-event-secrets
  annotations:
    kubernetes.io/service-account.name: "github-event-sa"
type: Opaque
data:
  DB_HOST: cG9zdGdyZXM=
```

### 3. Secret Rotation
```bash
# Script for secret rotation
#!/bin/bash
NEW_PASSWORD=$(openssl rand -base64 32)
kubectl patch secret github-event-secrets \
  --patch "{\"data\":{\"DB_PASSWORD\":\"$NEW_PASSWORD\"}}"
kubectl rollout restart deployment/api
kubectl rollout restart deployment/worker
```

### 4. Backup and Recovery
```bash
# Export secrets for backup
kubectl get secret github-event-secrets -o json > secrets-backup.json

# Restore from backup
kubectl apply -f secrets-backup.json
```

## Security Considerations

### 1. RBAC Setup
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: secret-reader-binding
subjects:
- kind: ServiceAccount
  name: github-event-sa
roleRef:
  kind: Role
  name: secret-reader
```

### 2. Secret Encryption
- Secrets are automatically encrypted at rest in etcd
- Use TLS for secret access from external systems
- Consider using external secret management for production

### 3. Audit and Monitoring
```bash
# Monitor secret access
kubectl get events --field-selector involvedObject.kind=Secret

# Audit secret usage
kubectl auth can-i get secret github-event-secrets
```

## Troubleshooting

### Common Issues

#### 1. Secret Not Found
```bash
Error: error validating data: failed to validate data: [ValidationError]
```
**Solution**: Check secret exists and keys are correct

#### 2. Base64 Encoding Issues
```bash
Error: Secret "github-event-secrets" is invalid
```
**Solution**: Ensure values are properly base64 encoded

#### 3. Permission Denied
```bash
Error: error validating data: ValidationError(ServiceAccount)
```
**Solution**: Check RBAC permissions and service account

### Debug Commands
```bash
# Check if pod is using secrets
kubectl describe pod <pod-name> | grep -A 10 "Mounts:"

# Check secret injection
kubectl exec <pod-name> -- env | grep -E "(DB_|REDIS_)"

# Verify secret values
kubectl get secret github-event-secrets -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
```

## Deployment with Secrets

### 1. Apply Updated Manifests
```bash
# Apply all configurations
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/worker.yaml
kubectl apply -f k8s/frontend.yaml
```

### 2. Verify Secret Usage
```bash
# Check pods are using secrets
kubectl get pods -o wide

# Describe pod to see environment variables
kubectl describe deployment/api | grep -A 20 "Environment:"
```

### 3. Test Configuration
```bash
# Test API connectivity
kubectl port-forward service/api 8080:8080
curl http://localhost:8080/events

# Check worker logs
kubectl logs -f deployment/worker
```

## Production Considerations

### 1. External Secret Management
- Consider using AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault
- Implement secret rotation policies
- Use IAM roles for least privilege access

### 2. Multi-Environment Strategy
- Use different secrets per environment
- Implement CI/CD pipeline secret injection
- Use GitOps for secret management

### 3. Monitoring and Alerting
- Set up alerts for secret access
- Monitor secret rotation schedules
- Log secret modification attempts

## Migration from Environment Variables

### Step 1: Backup Current Config
```bash
# Save current environment variables
kubectl get deployment/api -o yaml > api-backup.yaml
kubectl get deployment/worker -o yaml > worker-backup.yaml
```

### Step 2: Create Secrets
```bash
# Create secrets from environment variables
kubectl create secret generic github-event-secrets \
  --from-literal=DB_HOST=postgres \
  --from-literal=DB_PORT=5432 \
  --from-literal=DB_NAME=github_events \
  --from-literal=DB_USER=postgres \
  --from-literal=DB_PASSWORD=postgres \
  --from-literal=REDIS_HOST=redis \
  --from-literal=REDIS_PORT=6379
```

### Step 3: Update Deployments
```bash
# Apply updated manifests with secret references
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/worker.yaml
```

### Step 4: Verify Migration
```bash
# Check deployments are using secrets
kubectl describe deployment/api | grep -A 15 "Environment:"
kubectl describe deployment/worker | grep -A 15 "Environment:"

# Test functionality
kubectl logs -f deployment/api
kubectl logs -f deployment/worker
```

This comprehensive guide covers all aspects of Kubernetes Secrets management for the GitHub Event System, from basic setup to advanced production considerations.
