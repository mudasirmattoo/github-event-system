# Kubernetes Deployment Guide - GitHub Event System

## ✅ Issue Resolution Summary

The worker deployment was failing due to Redis connection issues. The problem was that Kubernetes automatically creates multiple environment variables with protocol prefixes that conflicted with Redis client expectations.

### Root Cause
Kubernetes was creating environment variables like:
```
REDIS_PORT=tcp://10.103.252.116:6379  # ❌ Wrong format
REDIS_PORT_6379_TCP=tcp://10.103.252.116:6379  # ❌ Wrong format
```

### Solution Applied
Updated worker deployment to explicitly set correct Redis environment variables:
```yaml
env:
  - name: REDIS_HOST
    value: redis
  - name: REDIS_PORT
    value: "6379"  # ✅ Explicitly set to port number only
  - name: DB_HOST
    value: postgres
```

## Current Status
- ✅ **PostgreSQL**: Connected successfully
- ✅ **Redis**: Connected successfully  
- ✅ **Workers**: All 3 worker pods running
- ✅ **No connection errors**: System is stable

## Complete Deployment Commands

### 1. Deploy All Services
```bash
# Deploy database
kubectl apply -f k8s/postgres.yaml

# Deploy Redis
kubectl apply -f k8s/redis.yaml

# Deploy API
kubectl apply -f k8s/api.yaml

# Deploy Workers
kubectl apply -f k8s/worker.yaml
```

### 2. Check Deployment Status
```bash
# Check all pods
kubectl get pods

# Check specific services
kubectl get pods -l app=worker
kubectl get pods -l app=api
kubectl get pods -l app=postgres
kubectl get pods -l app=redis

# Check services
kubectl get services
```

### 3. Monitor Logs
```bash
# Worker logs
kubectl logs deployment/worker --tail=20

# API logs
kubectl logs deployment/api --tail=20

# PostgreSQL logs
kubectl logs deployment/postgres --tail=20

# Redis logs
kubectl logs deployment/redis --tail=20
```

### 4. Troubleshooting Commands

#### Check Environment Variables
```bash
# Check worker environment variables
kubectl exec deployment/worker -- env | grep -E "(REDIS|DB|POSTGRES)"

# Check specific pod environment
kubectl exec -it <pod-name> -- env | grep REDIS
```

#### Debug Redis Connection
```bash
# Test Redis connection from worker pod
kubectl exec deployment/worker -- redis-cli -h redis -p 6379 ping

# Test Redis service
kubectl exec deployment/redis -- redis-cli ping
```

#### Debug Database Connection  
```bash
# Test database connection from worker pod
kubectl exec deployment/worker -- psql -h postgres -U postgres -d github_events -c "SELECT 1;"

# Test database service
kubectl exec deployment/postgres -- psql -U postgres -d github_events -c "SELECT 1;"
```

### 5. Scaling and Updates

#### Scale Workers
```bash
# Scale to 5 workers
kubectl scale deployment worker --replicas=5

# Scale to 1 worker
kubectl scale deployment worker --replicas=1
```

#### Update Deployments
```bash
# Update worker with new image
kubectl set image deployment/worker github-worker=github-worker:v2

# Rolling restart
kubectl rollout restart deployment/worker

# Check rollout status
kubectl rollout status deployment/worker
```

### 6. Service Access

#### API Service
```bash
# Get API service URL
minikube service api --url

# Or port-forward
kubectl port-forward service/api 8080:8080
```

#### Frontend Service (if deployed)
```bash
# Get frontend service URL  
minikube service frontend --url

# Or port-forward
kubectl port-forward service/frontend 3000:80
```

## Key Configuration Files

### Worker Deployment (k8s/worker.yaml)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: worker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: worker
  template:
    metadata:
      labels:
        app: worker
    spec:
      containers:
      - name: worker
        image: github-worker
        imagePullPolicy: Never
        env:
        - name: REDIS_HOST
          value: redis
        - name: REDIS_PORT
          value: "6379"  # ✅ Explicit port number
        - name: DB_HOST
          value: postgres
```

### API Deployment (k8s/api.yaml)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: api
        image: github-api
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
        env:
        - name: REDIS_HOST
          value: redis
        - name: DB_HOST
          value: postgres
---
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  type: NodePort
  selector:
    app: api
  ports:
    - port: 8080
      nodePort: 30080
```

### Database and Redis Services
```yaml
# PostgreSQL (k8s/postgres.yaml)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:latest
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_DB
              value: "github_events"
            - name: POSTGRES_USER
              value: "postgres"
            - name: POSTGRES_PASSWORD
              value: "postgres"

# Redis (k8s/redis.yaml)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:latest
          ports:
            - containerPort: 6379
```

## Best Practices

### Environment Variables
- Always use explicit port numbers in Kubernetes deployments
- Avoid relying on automatically generated environment variables with protocol prefixes
- Use `SERVICE_PORT` variables when available

### Image Management
- Use versioned images in production
- Set `imagePullPolicy: IfNotPresent` for production
- Use `imagePullPolicy: Never` for local images with minikube

### Monitoring
- Set up proper resource limits and requests
- Configure liveness and readiness probes
- Use structured logging for better debugging

### Security
- Use secrets for sensitive data (passwords, API keys)
- Configure network policies
- Set appropriate resource limits

## Next Steps

1. **Test webhook integration** with ngrok
2. **Set up monitoring** with Prometheus/Grafana
3. **Configure ingress** for external access
4. **Set up CI/CD** for automated deployments
5. **Add health checks** to all deployments

The system is now fully operational in Kubernetes!
