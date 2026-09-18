# LoreLattice Helm Chart

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/lorelattice)](https://artifacthub.io/packages/helm/lorelattice/lorelattice)

Helm chart for deploying [LoreLattice](https://github.com/Pototoooo/lorelattice) - an AI-powered Knowledge RAG Platform.

## Overview

LoreLattice is an intelligent knowledge base platform that combines:
- Document parsing and understanding
- Vector search with BM25 hybrid retrieval
- LLM integration for conversational AI
- Multi-tenant support with encryption

## Prerequisites

- Kubernetes 1.25+
- Helm 3.10+
- PV provisioner support in the underlying infrastructure
- Ingress controller (nginx-ingress recommended) for external access

## Quick Start

```bash
# Add required secrets
helm install lorelattice ./helm \
  --namespace lorelattice \
  --create-namespace \
  --set secrets.dbPassword=<your-db-password> \
  --set secrets.redisPassword=<your-redis-password> \
  --set secrets.jwtSecret=<your-jwt-secret>
```

## Architecture

```
                    ┌─────────────┐
                    │   Ingress   │
                    └──────┬──────┘
                           │
           ┌───────────────┴───────────────┐
           │                               │
           ▼                               ▼
    ┌─────────────┐                 ┌─────────────┐
    │  Frontend   │                 │   Backend   │
    │  (Vue.js)   │                 │   (Go/Gin)  │
    └─────────────┘                 └──────┬──────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    │                      │                      │
                    ▼                      ▼                      ▼
             ┌─────────────┐        ┌─────────────┐        ┌─────────────┐
             │  Docreader  │        │  PostgreSQL │        │    Redis    │
             │   (gRPC)    │        │  (ParadeDB) │        │   (Queue)   │
             └─────────────┘        └─────────────┘        └─────────────┘
```

## Installation

### Basic Installation

```bash
helm install lorelattice ./helm \
  --namespace lorelattice \
  --create-namespace \
  --set secrets.dbPassword=secure-password \
  --set secrets.redisPassword=secure-password \
  --set secrets.jwtSecret=$(openssl rand -base64 32)
```

### With Ingress

```bash
helm install lorelattice ./helm \
  --namespace lorelattice \
  --create-namespace \
  --set ingress.enabled=true \
  --set ingress.host=lorelattice.example.com \
  --set ingress.tls.enabled=true \
  --set ingress.tls.secretName=lorelattice-tls \
  --set secrets.dbPassword=secure-password \
  --set secrets.redisPassword=secure-password \
  --set secrets.jwtSecret=$(openssl rand -base64 32)
```

### With External LLM (Ollama)

```bash
helm install lorelattice ./helm \
  --namespace lorelattice \
  --create-namespace \
  --set app.extraEnv[0].name=OLLAMA_BASE_URL \
  --set app.extraEnv[0].value=http://ollama.ollama:11434 \
  --set app.extraEnv[1].name=INIT_LLM_MODEL_NAME \
  --set app.extraEnv[1].value=qwen2.5:7b \
  --set secrets.dbPassword=secure-password \
  --set secrets.redisPassword=secure-password \
  --set secrets.jwtSecret=$(openssl rand -base64 32)
```

### Production Installation

For production, use a values file:

```yaml
# values-production.yaml
global:
  storageClass: "fast-ssd"

app:
  replicaCount: 3
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2
      memory: 4Gi

postgresql:
  persistence:
    size: 100Gi

ingress:
  enabled: true
  host: lorelattice.company.com
  tls:
    enabled: true
    secretName: lorelattice-tls

secrets:
  existingSecret: lorelattice-secrets  # Use pre-created secret
```

```bash
helm install lorelattice ./helm \
  --namespace lorelattice \
  --create-namespace \
  -f values-production.yaml
```

## Configuration

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.storageClass` | Storage class for PVCs | `""` |
| `global.imagePullSecrets` | Image pull secrets | `[]` |
| `global.podSecurityContext` | Pod security context | See values.yaml |
| `global.containerSecurityContext` | Container security context | See values.yaml |

### ServiceAccount

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create ServiceAccount | `true` |
| `serviceAccount.name` | ServiceAccount name | `""` |
| `serviceAccount.annotations` | ServiceAccount annotations | `{}` |

### App (Backend)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `app.enabled` | Enable backend | `true` |
| `app.replicaCount` | Number of replicas | `1` |
| `app.image.repository` | Image repository | `pototoooo/lorelattice-app` |
| `app.image.tag` | Image tag | `""` (uses appVersion) |
| `app.resources` | Resource limits | See values.yaml |
| `app.env` | Environment variables | See values.yaml |
| `app.extraEnv` | Additional env vars | `[]` |

### Frontend

| Parameter | Description | Default |
|-----------|-------------|---------|
| `frontend.enabled` | Enable frontend | `true` |
| `frontend.replicaCount` | Number of replicas | `1` |
| `frontend.image.repository` | Image repository | `pototoooo/lorelattice-ui` |
| `frontend.image.tag` | Image tag | `latest` |

### PostgreSQL (ParadeDB)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `postgresql.enabled` | Enable PostgreSQL | `true` |
| `postgresql.image.repository` | Image repository | `paradedb/paradedb` |
| `postgresql.image.tag` | Image tag | `v0.18.9-pg17` |
| `postgresql.persistence.enabled` | Enable persistence | `true` |
| `postgresql.persistence.size` | PVC size | `10Gi` |

### Redis

| Parameter | Description | Default |
|-----------|-------------|---------|
| `redis.enabled` | Enable Redis | `true` |
| `redis.image.repository` | Image repository | `redis` |
| `redis.image.tag` | Image tag | `7-alpine` |
| `redis.persistence.enabled` | Enable persistence | `true` |
| `redis.persistence.size` | PVC size | `1Gi` |

### Ingress

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class | `nginx` |
| `ingress.host` | Hostname | `lorelattice.example.com` |
| `ingress.tls.enabled` | Enable TLS | `false` |
| `ingress.tls.secretName` | TLS secret name | `""` |

### Secrets

| Parameter | Description | Default |
|-----------|-------------|---------|
| `secrets.dbUser` | Database username | `postgres` |
| `secrets.dbPassword` | Database password | `""` (required) |
| `secrets.dbName` | Database name | `lorelattice` |
| `secrets.redisPassword` | Redis password | `""` (required) |
| `secrets.jwtSecret` | JWT signing secret | `""` (required) |
| `secrets.existingSecret` | Use existing secret | `""` |

### Optional Components

These map to docker-compose profiles:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `minio.enabled` | Enable MinIO storage | `false` |
| `neo4j.enabled` | Enable Neo4j (GraphRAG) | `false` |
| `qdrant.enabled` | Enable Qdrant vector DB | `false` |

## Security Best Practices

### Secret Management

**Never commit secrets to Git!** Use one of these approaches:

1. **Helm --set flags** (for testing)
   ```bash
   helm install lorelattice ./helm --set secrets.dbPassword=xxx
   ```

2. **External Secrets Operator** (recommended for production)
   ```yaml
   secrets:
     existingSecret: lorelattice-external-secret
   ```

3. **Sealed Secrets** (for GitOps)
   ```bash
   kubeseal < secret.yaml > sealed-secret.yaml
   ```

### Pod Security

The chart follows CNCF security best practices:
- Runs as non-root user
- Read-only root filesystem where possible
- Drops all capabilities
- Uses seccomp profiles

## Upgrading

```bash
helm upgrade lorelattice ./helm \
  --namespace lorelattice \
  --reuse-values
```

## Uninstalling

```bash
helm uninstall lorelattice --namespace lorelattice

# Optional: Remove PVCs
kubectl delete pvc -n lorelattice -l app.kubernetes.io/instance=lorelattice
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods -n lorelattice
```

### View Logs
```bash
# Backend logs
kubectl logs -n lorelattice -l app.kubernetes.io/component=app -f

# Frontend logs
kubectl logs -n lorelattice -l app.kubernetes.io/component=frontend -f
```

### Common Issues

**Pod stuck in Pending**
- Check if PVCs are bound: `kubectl get pvc -n lorelattice`
- Verify storage class exists: `kubectl get sc`

**Connection refused errors**
- Wait for all pods to be Ready
- Check service endpoints: `kubectl get endpoints -n lorelattice`

**Database connection errors**
- Verify secrets are correct
- Check PostgreSQL logs: `kubectl logs -n lorelattice -l app.kubernetes.io/component=database`

## Contributing

See [CONTRIBUTING.md](https://github.com/Pototoooo/lorelattice/blob/main/CONTRIBUTING.md) in the main repository.

## References

This Helm chart follows best practices from:
- [Helm Best Practices](https://helm.sh/docs/chart_best_practices/)
- [ArgoCD Helm Chart](https://github.com/argoproj/argo-helm)
- [Prometheus Helm Charts](https://github.com/prometheus-community/helm-charts)
- [cert-manager Helm Chart](https://github.com/cert-manager/cert-manager)

