# GitOps Deployment (ArgoCD)

> **Status: prepared, not activated.** Per [ADR 006](../adr/006-deployment-strategy.md)
> the MVP deploys as a single-host `docker-compose` stack. This directory is the
> ready-to-use Kubernetes story for the day traffic justifies a cluster.

## Layout

```
infra/
├── argocd/
│   ├── application-set.yaml   # one ArgoCD Application per overlay (dev, prod)
│   └── README.md
└── kustomize/
    ├── base/                  # backend + web Deployments/Services, ConfigMap
    └── overlays/
        ├── dev/               # 1 replica, :dev image tags
        └── prod/              # 2 replicas, :main image tags, bigger limits
```

## How it would work

1. **CI builds immutable images** (`ghcr.io/pudimproductivity/backend:<sha>`)
   instead of building on the target host.
2. **ArgoCD watches this repo.** The ApplicationSet renders one Application per
   overlay; each points at the matching Kustomize path. Prod auto-syncs on
   commit; dev syncs manually.
3. **Secrets** are injected with sealed-secrets or ExternalSecrets — never
   committed (see `docs/security/secrets-management.md`).
4. **Rollback** is `git revert` + the controller reverting the cluster state.

## Activation checklist (when a cluster exists)

- [ ] Push image build+push to CI (backend + web workflows).
- [ ] Install ArgoCD (`kubectl create namespace argocd && kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/<ver>/manifests/install.yaml`).
- [ ] Create the `pudimproductivity` project + apply the ApplicationSet.
- [ ] Provision the DB (managed RDS) and create the `backend-secrets` secret.
- [ ] Ingress: TLS cert + a single `Ingress` routing `/` → web, `/api/*` → backend (WebSocket upgrades need `nginx.ingress.kubernetes.io/proxy-read-timeout`).
- [ ] Consider `argocd-image-updater` for automatic rollout on new image tags
      (canary via Argo Rollouts once prod has real traffic).
