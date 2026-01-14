# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the Terraform Registry.

## Prerequisites

- Kubernetes cluster (v1.25+)
- kubectl configured to access your cluster
- Nginx Ingress Controller (for ingress)
- Storage class available for PVC (or use default)

## Quick Start

### 1. Deploy with kubectl

```bash
# Apply all resources
kubectl apply -k k8s/

# Or apply individually
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/backend-deployment.yaml
kubectl apply -f k8s/backend-service.yaml
kubectl apply -f k8s/frontend-deployment.yaml
kubectl apply -f k8s/frontend-service.yaml
kubectl apply -f k8s/ingress.yaml
```

### 2. Verify Deployment

```bash
# Check all resources
kubectl get all -n terraform-registry

# Check pods are running
kubectl get pods -n terraform-registry

# Check services
kubectl get svc -n terraform-registry

# Check ingress
kubectl get ingress -n terraform-registry
```

### 3. Access the Application

- **With Ingress**: Access via your configured domain (e.g., `https://registry.example.com`)
- **Port Forward** (for testing):
  ```bash
  # Frontend
  kubectl port-forward svc/terraform-registry-frontend 3000:80 -n terraform-registry
  
  # Backend API
  kubectl port-forward svc/terraform-registry-backend 8080:8080 -n terraform-registry
  ```

## Configuration

### Update Domain

Edit `ingress.yaml` and change `registry.example.com` to your actual domain.

### Update Secret

**Important**: Change the `AUTH_SECRETKEY` in `secret.yaml` before deploying to production!

```bash
# Generate a secure random key
openssl rand -base64 32

# Update the secret
kubectl create secret generic terraform-registry-secret \
  --from-literal=AUTH_SECRETKEY='your-secure-key-here' \
  -n terraform-registry \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Enable TLS

1. Create TLS secret:
   ```bash
   kubectl create secret tls terraform-registry-tls \
     --cert=path/to/tls.crt \
     --key=path/to/tls.key \
     -n terraform-registry
   ```

2. Uncomment TLS section in `ingress.yaml`

### Using cert-manager

If you have cert-manager installed:

1. Uncomment the cert-manager annotation in `ingress.yaml`
2. Uncomment the TLS section and set your domain

### Adjust Storage

Edit `pvc.yaml` to change:
- Storage size (default: 10Gi)
- Storage class (uncomment and set `storageClassName`)

### Adjust Resources

Edit deployment files to adjust CPU/memory requests and limits based on your needs.

## Architecture

```
                    ┌─────────────────────────────────────────────────────┐
                    │                    Ingress                          │
                    │              (registry.example.com)                 │
                    └─────────────────────────────────────────────────────┘
                                          │
                    ┌─────────────────────┼─────────────────────┐
                    │                     │                     │
              /api, /v1, /health         /                     
              /.well-known               (other)
                    │                     │
                    ▼                     ▼
         ┌──────────────────┐  ┌──────────────────┐
         │  Backend Service │  │ Frontend Service │
         │    (ClusterIP)   │  │    (ClusterIP)   │
         │     :8080        │  │       :80        │
         └──────────────────┘  └──────────────────┘
                    │                     │
                    ▼                     ▼
         ┌──────────────────┐  ┌──────────────────┐
         │ Backend Deploy   │  │ Frontend Deploy  │
         │   (1 replica)    │  │   (2 replicas)   │
         └──────────────────┘  └──────────────────┘
                    │
                    ▼
         ┌──────────────────┐
         │       PVC        │
         │   (10Gi data)    │
         └──────────────────┘
```

## Scaling

### Frontend

Frontend is stateless and can be scaled horizontally:

```bash
kubectl scale deployment terraform-registry-frontend --replicas=3 -n terraform-registry
```

### Backend

**Note**: Backend uses SQLite with local storage. For horizontal scaling, consider:
1. Switching to PostgreSQL/MySQL
2. Using shared storage (NFS, etc.)
3. Using StatefulSet instead of Deployment

## Troubleshooting

### Check Logs

```bash
# Backend logs
kubectl logs -l app.kubernetes.io/component=backend -n terraform-registry

# Frontend logs
kubectl logs -l app.kubernetes.io/component=frontend -n terraform-registry
```

### Check Events

```bash
kubectl get events -n terraform-registry --sort-by='.lastTimestamp'
```

### Debug Pod

```bash
# Exec into backend pod
kubectl exec -it deploy/terraform-registry-backend -n terraform-registry -- sh

# Exec into frontend pod
kubectl exec -it deploy/terraform-registry-frontend -n terraform-registry -- sh
```

## Cleanup

```bash
# Delete all resources
kubectl delete -k k8s/

# Or delete namespace (removes everything)
kubectl delete namespace terraform-registry
```
