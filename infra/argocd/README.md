# GitOps Deployment (ArgoCD)

> **Status: deployed on a local Kind cluster (2026-08-10).** The public-GitHub
> ApplicationSet (`application-set.yaml`) is validated but not applied — the
> local-demo Application in `local-demo/` proves the full GitOps loop
> (commit → local git daemon → ArgoCD auto-sync → Healthy) against the same
> Kustomize overlays. Production activation is still gated on a reachable
> cluster + registry per [ADR 006](../adr/006-deployment-strategy.md).

## Layout

```
infra/
├── argocd/
│   ├── application-set.yaml   # one ArgoCD Application per overlay (dev, prod)
│   ├── local-demo/            # Kind-cluster demo: local git daemon repo + Application
│   └── README.md
├── kind/
│   └── kind-config.yaml       # single-node cluster with NodePort mappings
└── kustomize/
    ├── base/                  # backend + web Deployments/Services, ConfigMap
    └── overlays/
        ├── dev/               # 1 replica, :dev image tags, in-cluster Postgres
        └── prod/              # 2 replicas, :main image tags, bigger limits
```

## Deploying the local demo (Kind + ArgoCD)

1. **Boot the cluster.**

   ```sh
   kind create cluster --config infra/kind/kind-config.yaml
   ```

2. **Build and load images** (the dev overlay uses `pudim/backend:dev` /
   `pudim/web:dev`, loaded directly into the cluster — no registry needed).

   ```sh
   docker build -t pudim/backend:dev -f backend/Dockerfile backend/
   docker build -t pudim/web:dev -f web/Dockerfile web/
   kind load docker-image --name pudim pudim/backend:dev pudim/web:dev
   ```

3. **Install ArgoCD.**

   ```sh
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.14.0/manifests/install.yaml
   kubectl wait --for=condition=available deployment/argocd-server -n argocd --timeout=180s
   ```

4. **Serve the repo over a local git daemon** so ArgoCD can clone without
   pushing to GitHub.

   ```sh
   git clone --mirror "$PWD" "$HOME/git-server/pudim.git"
   git daemon --base-path=$HOME/git-server --export-all --reuseaddr --detach
   # From inside the cluster, the host is reachable at the Docker bridge
   # gateway (172.19.0.1 for a default Kind network).
   ```

5. **Register the repo + Application** (auto-sync, prune, self-heal).

   ```sh
   kubectl apply -f infra/argocd/local-demo/repository.yaml
   kubectl apply -f infra/argocd/local-demo/application.yaml
   ```

6. **Verify.** The Application should report `Synced / Healthy`:

   ```sh
   kubectl get application -n argocd pudimproductivity-dev \
     -o jsonpath='{.status.sync.status} / {.status.health.status}'
   kubectl get pods -n pudimproductivity   # backend, web, postgres all Running
   ```

## Production activation checklist

- [x] Kustomize overlays verified against a real cluster (local demo).
- [x] ArgoCD install + repo registration + Application sync proven locally.
- [ ] Push image build+push to CI (backend + web workflows) — images are
      currently loaded directly into the demo cluster.
- [ ] Apply `application-set.yaml` once the GitHub repo is reachable from the
      cluster (validated with `kubectl apply --dry-run=client`).
- [ ] Provision the DB (managed RDS, see `infra/terraform/`) and create the
      `backend-secrets` Secret.
- [ ] Ingress: TLS cert + a single `Ingress` routing `/` → web, `/api/*` →
      backend (WebSocket upgrades need
      `nginx.ingress.kubernetes.io/proxy-read-timeout`).
- [ ] Consider `argocd-image-updater` for automatic rollout on new image tags
      (canary via Argo Rollouts once prod has real traffic).

