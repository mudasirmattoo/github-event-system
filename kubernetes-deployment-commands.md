# Kubernetes End-to-End Deployment Commands

## Prerequisites
- Minikube installed and running
- Docker installed and running
- kubectl configured to use minikube

## Step 1: Start Minikube
```bash
# Start minikube
minikube start

# Enable ingress (optional, for external access)
minikube addons enable ingress

# Set docker environment to use minikube's docker daemon
eval $(minikube docker-env)
```

## Step 2: Build Docker Images
```bash
# Build API image
cd api
docker build -t github-api .

# Build Worker image
cd ../worker
docker build -t github-worker .

# Go back to project root
cd ..
```

## Step 3: Deploy Infrastructure Services
```bash
# Deploy PostgreSQL ConfigMap (database schema)
kubectl apply -f k8s/postgres-configmap.yaml

# Deploy PostgreSQL
kubectl apply -f k8s/postgres.yaml

# Deploy Redis
kubectl apply -f k8s/redis.yaml

# Wait for infrastructure to be ready
kubectl wait --for=condition=ready pod -l app=postgres --timeout=60s
kubectl wait --for=condition=ready pod -l app=redis --timeout=60s
```

## Step 4: Deploy Application Services
```bash
# Deploy API
kubectl apply -f k8s/api.yaml

# Deploy Workers
kubectl apply -f k8s/worker.yaml

# Wait for application to be ready
kubectl wait --for=condition=ready pod -l app=api --timeout=60s
kubectl wait --for=condition=ready pod -l app=worker --timeout=60s
```

## Step 5: Verify Deployment
```bash
# Check all pods
kubectl get pods

# Check all services
kubectl get services

# Check deployment status
kubectl get deployments

# Check detailed pod information
kubectl describe pods
```

## Step 6: Test the System
```bash
# Test API service
minikube service api --url

# Port forward API for local testing
kubectl port-forward service/api 8080:8080

# Test API endpoint
curl http://localhost:8080/api/events

# Test webhook endpoint
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test":"webhook"}'
```

## Step 7: Monitor Logs
```bash
# Monitor all services logs
kubectl logs -l app=api --tail=20
kubectl logs -l app=worker --tail=20
kubectl logs -l app=postgres --tail=20
kubectl logs -l app=redis --tail=20

# Follow logs in real-time
kubectl logs -l app=worker -f
```

## Step 8: Scale Services (Optional)
```bash
# Scale workers to 5 replicas
kubectl scale deployment worker --replicas=5

# Scale API to 2 replicas
kubectl scale deployment api --replicas=2

# Check scaling status
kubectl get pods -l app=worker
kubectl get pods -l app=api
```

## Step 9: Clean Up (When Done)
```bash
# Delete all deployments
kubectl delete -f k8s/

# Stop minikube
minikube stop

# Delete minikube cluster (optional)
minikube delete
```

## Troubleshooting Commands

### Check Pod Issues
```bash
# Get pod events
kubectl describe pod <pod-name>

# Get pod logs
kubectl logs <pod-name>

# Get pod logs with previous container
kubectl logs <pod-name> --previous

# Exec into pod for debugging
kubectl exec -it <pod-name> -- /bin/bash
```

### Check Service Issues
```bash
# Test service connectivity
kubectl exec -it <pod-name> -- curl http://api:8080/api/events

# Check service endpoints
kubectl get endpoints

# Check service details
kubectl describe service <service-name>
```

### Database and Redis Debug
```bash
# Test database connection
kubectl exec -it deployment/postgres -- psql -U postgres -d github_events -c "SELECT 1;"

# Test Redis connection
kubectl exec -it deployment/redis -- redis-cli ping

# Check Redis from worker pod
kubectl exec -it deployment/worker -- redis-cli -h redis -p 6379 ping
```

### Check Environment Variables
```bash
# Check environment variables in pods
kubectl exec deployment/api -- env | grep -E "(REDIS|DB)"
kubectl exec deployment/worker -- env | grep -E "(REDIS|DB)"
```

## Expected Results

### Successful Deployment Should Show:
```bash
# kubectl get pods
NAME                      READY   STATUS    RESTARTS   AGE
postgres-xxxxx            1/1     Running   0          2m
redis-xxxxx               1/1     Running   0          2m
api-xxxxx                 1/1     Running   0          1m
worker-xxxxx              1/1     Running   0          1m
worker-xxxxx              1/1     Running   0          1m
worker-xxxxx              1/1     Running   0          1m

# kubectl get services
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
postgres     ClusterIP   10.96.0.1       <none>        5432/TCP        2m
redis        ClusterIP   10.96.0.2       <none>        6379/TCP        2m
api          NodePort    10.96.0.3       <none>        8080:30080/TCP  1m
```

### API Test Should Return:
```bash
curl http://localhost:8080/api/events
[{"branch":"main","created_at":"2026-04-19T16:36:23.871576Z","delivery_id":"sample-delivery-1","event_type":"push","message":"Initial commit","repo":"test-repo","retry_count":0,"status":"success"},...]
```

### Worker Logs Should Show:
```bash
kubectl logs deployment/worker --tail=10
2026/04/20 12:00:00 connected to postgres
2026/04/20 12:00:00 Worker started... Waiting for events
2026/04/20 12:00:00 Worker 0 started
2026/04/20 12:00:00 Worker 1 started
2026/04/20 12:00:00 Worker 2 started
```

## Notes
- Ensure `eval $(minikube docker-env)` is set before building images
- Use `imagePullPolicy: Never` for local images
- API is accessible via NodePort 30080 or `minikube service api --url`
- Workers automatically connect to Redis and PostgreSQL services
- All services use proper Kubernetes DNS for inter-service communication
