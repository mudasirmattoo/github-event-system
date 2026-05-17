# AWS EKS & ECR Deployment Guide

This guide outlines the step-by-step process of migrating the **GitHub Event System** from a local Minikube environment to **AWS EKS** (Elastic Kubernetes Service) and **AWS ECR** (Elastic Container Registry).

---

## 1. Architecture Overview

The system consists of the following components running in AWS:
* **EKS Cluster**: `github-event-system-eks` (Kubernetes version `1.31`, running on managed node groups in private subnets).
* **ECR Registry**: `537765357717.dkr.ecr.us-east-1.amazonaws.com`
* **Single ECR Repository**: `github-events-repo` (uses tag suffixes to distinguish the microservices).
* **VPC & Subnets**: A modular VPC with dedicated public subnets (hosting an internet-facing NAT gateway) and private subnets (hosting EKS worker nodes).

---

## 2. Step-by-Step Deployment Walkthrough

### Step 1: Authenticate Local Docker with AWS ECR
Before you can tag and push docker images, your local Docker daemon must authenticate with your private AWS ECR registry:
```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 537765357717.dkr.ecr.us-east-1.amazonaws.com
```

---

### Step 2: Tag & Push Local Docker Images
Since we are using a **single ECR repository** (`github-events-repo`) to store all three microservice images, we tag them using distinct image tag suffixes (`:api-latest`, `:worker-latest`, `:frontend-latest`).

#### Tag local images:
```bash
# Tag the API service
docker tag github-event-api:latest 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:api-latest

# Tag the Worker service
docker tag github-event-worker:latest 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:worker-latest

# Tag the Frontend service
docker tag github-event-frontend:latest 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:frontend-latest
```

#### Push tagged images to ECR:
```bash
# Push the API image
docker push 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:api-latest

# Push the Worker image
docker push 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:worker-latest

# Push the Frontend image
docker push 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:frontend-latest
```

---

### Step 3: Switch Kubectl Context to EKS
Configure your local `kubectl` to interact with your AWS EKS cluster. This will fetch cluster credentials and set EKS as your active context:
```bash
aws eks update-kubeconfig --region us-east-1 --name github-event-system-eks
```

#### Verify EKS Connectivity:
Confirm that you can communicate with EKS and that your EKS worker nodes are healthy and active:
```bash
# Verify active context
kubectl config current-context

# List EKS nodes
kubectl get nodes
```
*Expected Output:*
```text
NAME                         STATUS   ROLES    AGE    VERSION
ip-10-0-1-113.ec2.internal   Ready    <none>   151m   v1.31.14-eks-7fcd7ec
ip-10-0-2-125.ec2.internal   Ready    <none>   151m   v1.31.14-eks-7fcd7ec
```

---

### Step 4: Provision Kubernetes Secrets on EKS
The API and Worker services expect database and caching credentials to be stored inside a Kubernetes Secret named `github-event-secrets`. Create this secret in the cluster:
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

---

### Step 5: Update Kubernetes Manifests
To allow EKS to pull container images from AWS ECR, ensure that your local deployment manifests (`k8s/`) use the correct ECR registry URLs and set `imagePullPolicy: IfNotPresent` (or `Always`), as remote clusters cannot load images from your local host daemon.

#### API Deployment (`k8s/api.yaml`):
```yaml
      containers:
      - name: api
        image: 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:api-latest
        imagePullPolicy: IfNotPresent
```

#### Worker Deployment (`k8s/worker.yaml`):
```yaml
      containers:
      - name: worker
        image: 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:worker-latest
        imagePullPolicy: IfNotPresent
```

#### Frontend Deployment (`k8s/frontend.yaml`):
```yaml
      containers:
      - name: frontend
        image: 537765357717.dkr.ecr.us-east-1.amazonaws.com/github-events-repo:frontend-latest
        imagePullPolicy: IfNotPresent
```

---

### Step 6: Deploy all Resources to EKS
Apply all Kubernetes manifests (PostgreSQL schema initialization scripts, database stateful resources, Redis cache cluster, and microservice deployments/services) to the EKS cluster:
```bash
# Apply all manifests in the k8s/ directory
kubectl apply -f k8s/
```

---

### Step 7: Verify Deployments & Pod Status
Verify that all deployments are created and that all pods are healthy and in the `Running` state:
```bash
# List all pods
kubectl get pods
```
*Expected healthy state output:*
```text
NAME                        READY   STATUS    RESTARTS   AGE
api-6c949cd5c6-6l7g8        1/1     Running   0          46s
frontend-58fcc97c99-8pfvj   1/1     Running   0          45s
postgres-6db55ccb4f-47bhl   1/1     Running   0          43s
redis-f8db9547-75bqg        1/1     Running   0          42s
worker-7dbc88b9df-4m55r     1/1     Running   0          41s
worker-7dbc88b9df-szgw5     1/1     Running   0          41s
worker-7dbc88b9df-xnl2s     1/1     Running   0          41s
```

---

## 3. Local Port Forwarding & Testing

Because EKS worker nodes run inside secure private subnets, your microservices are not immediately accessible via public IP addresses. To access the dashboard and send webhooks locally, use port forwarding:

### 1. Access Frontend Dashboard
Forward the EKS frontend service (port `80`) to port `3000` on your localhost:
```bash
kubectl port-forward service/frontend 3000:80
```
Open [http://localhost:3000](http://localhost:3000) in your browser to view your live event monitoring dashboard.

### 2. Access API Service
Forward the EKS API service (port `8080`) to port `8080` on your localhost:
```bash
kubectl port-forward service/api 8080:8080
```
You can now test endpoints or forward webhook deliveries to [http://localhost:8080/webhook](http://localhost:8080/webhook).

---

## 4. Useful Debugging & Maintenance Commands

### Stream logs in real-time:
```bash
# Monitor API logs
kubectl logs -f deployment/api --tail=50

# Monitor Worker logs
kubectl logs -f deployment/worker --tail=50
```

### Restart a deployment (e.g. after pushing new docker updates):
```bash
# Rollout restart EKS pods to pull the latest image from ECR
kubectl rollout restart deployment/api
kubectl rollout restart deployment/worker
kubectl rollout restart deployment/frontend
```
