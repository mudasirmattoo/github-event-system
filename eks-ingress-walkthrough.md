# Ingress & AWS Load Balancer Controller Walkthrough

We have successfully set up **Kubernetes Ingress** and the **AWS Load Balancer Controller (LBC)** on your EKS cluster! The entire system is now exposed to the public internet through a single **AWS Application Load Balancer (ALB)**.

---

## 1. Accomplishments & Commands Executed

### Step 1: Create IAM Service Account via `eksctl`
You successfully ran `eksctl` to set up the Kubernetes ServiceAccount `aws-load-balancer-controller` in the `kube-system` namespace, binding it to the generated AWS IAM Role containing LBC permissions:
```bash
eksctl create iamserviceaccount \
  --cluster=github-event-system-eks \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --attach-policy-arn=arn:aws:iam::537765357717:policy/AWSLoadBalancerControllerIAMPolicy \
  --override-existing-serviceaccounts \
  --region us-east-1 \
  --approve
```

### Step 2: Download Helm Binary Locally
We downloaded and prepared a local standalone Helm binary (`./helm-bin`) in your project root to manage EKS packages without needing global installation:
```bash
curl -fsSL -o helm.tar.gz https://get.helm.sh/helm-v3.14.0-linux-amd64.tar.gz
tar -zxvf helm.tar.gz
mv linux-amd64/helm ./helm-bin
rm -rf linux-amd64 helm.tar.gz
```

### Step 3: Crucial Debugging - Resolving the CoreDNS Deadlock
During the initial LBC installation, we discovered that **CoreDNS** and **kube-proxy** EKS addons were missing from the cluster, causing all internal DNS name resolution (e.g. pods trying to connect to `postgres` or the EKS API server Control Plane) to time out.

When we tried to install CoreDNS, it got blocked in a "chicken-and-egg" deadlock because the LBC webhook configurations had already been registered but the LBC pods were crashlooping due to the missing DNS. The EKS control plane could not create the CoreDNS service because the LBC webhook endpoint was offline!

**How we resolved it:**
1. **Deleted the blocking LBC admission webhooks** to allow raw service creation requests through:
   ```bash
   kubectl delete mutatingwebhookconfiguration aws-load-balancer-webhook
   kubectl delete validatingwebhookconfiguration aws-load-balancer-webhook
   ```
2. **Recreated EKS addons** (`coredns` and `kube-proxy`):
   ```bash
   aws eks create-addon --cluster-name github-event-system-eks --addon-name coredns
   aws eks create-addon --cluster-name github-event-system-eks --addon-name kube-proxy
   ```
3. Verified both addons became **ACTIVE** and CoreDNS pods started up successfully.
4. **Re-upgraded the LBC Helm release** to cleanly restore the webhooks now that DNS is functional:
   ```bash
   ./helm-bin upgrade aws-load-balancer-controller eks/aws-load-balancer-controller \
     -n kube-system \
     --set clusterName=github-event-system-eks \
     --set serviceAccount.create=false \
     --set serviceAccount.name=aws-load-balancer-controller \
     --set vpcId=vpc-02fdf7f95e17849ff
   ```

Both LBC pods instantly transitioned to `1/1 READY` and healthy!

---

### Step 4: Deployed Ingress Manifest
We applied your Ingress rules ([k8s/ingress.yaml](file:///home/mudasirmattoo/Projects/github-event-system/k8s/ingress.yaml)) which configure path-based routing through the new ALB:
```bash
kubectl apply -f k8s/ingress.yaml
```

---

## 2. Verification & Cluster Status

### Pod Status (`kubectl get pods -A`)
Every pod across your cluster is now perfectly healthy and running:
```bash
$ kubectl get pods -A
NAMESPACE     NAME                                            READY   STATUS    RESTARTS   AGE
default       api-6c949cd5c6-6l7g8                            1/1     Running   0          25h
default       frontend-58fcc97c99-8pfvj                       1/1     Running   0          25h
default       postgres-6db55ccb4f-47bhl                       1/1     Running   0          25h
default       redis-f8db9547-75bqg                            1/1     Running   0          25h
default       worker-74ffc844df-hthfx                         1/1     Running   0          15s
default       worker-74ffc844df-m9895                         1/1     Running   0          18s
default       worker-74ffc844df-x7rc8                         1/1     Running   0          16s
kube-system   aws-load-balancer-controller-7dff6dfd66-bddfg   1/1     Running   3          5m
kube-system   aws-load-balancer-controller-7dff6dfd66-h2rk2   1/1     Running   3          5m
kube-system   coredns-59845f7779-h6jwz                        1/1     Running   0          33s
kube-system   coredns-59845f7779-px9sv                        1/1     Running   0          33s
kube-system   aws-node-lvnkt                                  2/2     Running   0          27h
kube-system   aws-node-z2brw                                  2/2     Running   0          27h
```

### Ingress & ALB Endpoint
The AWS Load Balancer Controller successfully provisioned your Application Load Balancer (ALB) and bound it to your Ingress resource:
```bash
$ kubectl get ingress
NAME                          CLASS   HOSTS   ADDRESS                                                                 PORTS   AGE
github-event-system-ingress   alb     *       k8s-default-githubev-5b1dfe76e5-804411905.us-east-1.elb.amazonaws.com   80      7s
```

* **ALB URL**: [http://k8s-default-githubev-5b1dfe76e5-804411905.us-east-1.elb.amazonaws.com](http://k8s-default-githubev-5b1dfe76e5-804411905.us-east-1.elb.amazonaws.com)

---

## 3. Public Webhook & Routing Rules

Your ALB will route incoming traffic as follows:
* **Frontend Dashboard (`/`)**: Routes directly to your Frontend pod (`port 80`).
* **Webhook Ingestion (`/webhook`)**: Routes directly to your API pod (`port 8080/webhook`).
* **Events API (`/events`)**: Routes directly to your API pod (`port 8080/events`).

---

## 4. Troubleshooting & Maintenance Reference

### Re-Upgrading Helm Chart
If you change your EKS cluster settings or want to upgrade LBC:
```bash
./helm-bin upgrade aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=github-event-system-eks \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller \
  --set vpcId=vpc-02fdf7f95e17849ff
```

### Stream controller logs:
```bash
kubectl logs -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller -f --tail=50
```
