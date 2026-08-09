# ADR 006: Deployment Strategy — Single-Host Compose with GitOps as Forward Path

**Status:** Accepted
**Date:** 2026-08-09
**Author:** Diego Braga

## Context

The product is a modular-monolith backend + web SPA + Android app. The MVP must
be deployable with minimal operational burden (a personal productivity app on a
LAN server / single VM) while the portfolio must show production-grade
deployment thinking: reproducible, reviewable, automated.

Options considered:

1. **Kubernetes + ArgoCD now** — maximum "enterprise" signal, but a real cluster
   for a single-user app is overkill: control-plane cost, certificate/ingress
   complexity, and no actual traffic to justify it.
2. **docker-compose on one host** — fits the workload, cheap, and the current
   repo already runs this for local dev.
3. **Platform-as-a-Service (Fly.io / Render)** — simplest ops, but pushes
   infrastructure outside the repo and reduces the IaC/CI story.

## Decision

**The MVP production deployment is a single-host docker-compose stack
(PostgreSQL + backend + web/nginx + RabbitMQ), deployed from CI by SSH + `docker
compose pull && up -d`.** GitOps (ArgoCD/Kustomize) is **not wired to a live
cluster**; instead the repository carries the manifests and GitOps semantics
that make the migration mechanical, documented under `infra/argocd/` and
`infra/kustomize/`.

Rationale:

- The **SLOs** (`docs/slo.md`) are achievable on a single host at MVP scale;
  there is no multi-node requirement yet.
- A compose stack is **fully reviewable in code** (infra as code at the right
  level), reproducible via CI, and cheap to run.
- Locking in k8s before real traffic exists would violate the cost-awareness
  requirement (Phase 10) and add operational burden without user value.
- ArgoCD-style **GitOps principles still apply**: the deployable state is
  described in the repo, CI produces immutable artifacts (tagged images), and
  rollback is `git revert` + redeploy.

## GitOps Semantics (as-if)

Even without a cluster, the repository treats deployable state as code:

- `docker-compose.yml` pins service definitions; environment secrets come from
  `.env` (never committed, see `docs/security/secrets-management.md`).
- CI builds tagged images (`ghcr.io/<owner>/pudim-backend:<sha>` in the future)
  instead of building on the target host, keeping deploy and build separate.
- `infra/kustomize/` (dev/staging/prod overlays) mirrors what an ArgoCD
  `ApplicationSet` would target — an honest, ready-to-use k8s story that is
  deliberately not activated until a cluster exists.

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **K8s + ArgoCD now** | Strongest deployment story | Cost + ops burden with zero traffic; over-engineered |
| **Fly.io / Render PaaS** | Cheapest ops | Infra lives outside the repo; weaker IaC narrative |
| **Manual SSH + git pull** | Trivial | Not reproducible, no artifact immutability |

## Consequences

**Positive:**

- Cheap, reproducible, reviewable deployment aligned with current scale.
- CI gains a deploy step (tagged image → target host) without a cluster.
- ArgoCD manifests + overlays exist so a later migration is mechanical.

**Negative:**

- No canary/rollout automation on a live cluster (documented as future work in
  Phase 10 checklist; `Flagger`/`Argo Rollouts` requires k8s).
- Single host is a single point of failure — accepted at MVP scale; the
  failover runbook (`docs/runbooks/db-pool-exhaustion.md`, postmortem 002)
  covers restore-on-hardware-failure.

## When to Revisit

- **Traffic/scale trigger:** sustained concurrency beyond what a single host
  can serve, or a requirement for horizontal autoscaling → activate
  `infra/argocd/` against a managed k8s cluster.
- **Multi-user (Phase 8):** session/connection state becomes distributed →
  cluster or managed broker required.
- **Availability requirement above 99.5%:** single-host topology cannot meet it;
  revisit with a two-node active/passive posture.
