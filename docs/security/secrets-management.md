# Secrets Management & Rotation

> This document describes how secrets (database passwords, API keys, signing keys) are managed across environments, and the procedures for rotating them securely.

---

## Current Approach (Local Development)

### Environment Variables

All secrets are loaded from a `.env` file at the repository root. This file is:

- **Never committed** to version control (listed in `.gitignore`).
- **Copied from `.env.example`** by each developer and filled with real values.
- **Read by `docker compose`** automatically — services reference `${VAR_NAME}` in `docker-compose.yml`.

### Current Secrets

| Secret | Environment Variable | Source | Rotation |
|--------|--------------------|--------|----------|
| PostgreSQL password | `POSTGRES_PASSWORD` | `.env` | N/A (local only) |
| RabbitMQ password | `RABBITMQ_PASS` | `.env` | N/A (local only) |
| Database URL | `DATABASE_URL` | Derived from POSTGRES_* vars | N/A (local only) |

### Local Security Notes

- Default passwords (e.g., `change_me_in_production`) **must be overridden** before any non-local deployment.
- Postgres exposes port `5433` on the host (not `5432`) to avoid conflicts with local PostgreSQL instances.
- RabbitMQ management UI (`15672`) should only be accessible on localhost.

---

## Production Approach (Planned)

### Secret Storage: AWS Secrets Manager

| Secret | Storage | Access Method |
|--------|---------|---------------|
| DB master password | AWS Secrets Manager | EKS pod IAM role (IRSA) |
| DB connection URL | AWS Secrets Manager | Generated from master password + host |
| JWT signing key | AWS Secrets Manager | EKS pod IAM role |
| FCM server key | AWS Secrets Manager | EKS pod IAM role |
| Google OAuth client secret | AWS Secrets Manager | EKS pod IAM role |

### Architecture

```
EKS Pod (backend)
  ↕ (Secrets Store CSI Driver)
AWS Secrets Manager
  ↕ (IAM Role — IRSA)
IAM Role (secrets-reader)
```

The **Secrets Store CSI Driver** mounts secrets as volumes or environment variables inside pods, without them being stored in etcd.

### Alternative: Kubernetes Native Secrets (Dev/Staging Only)

For dev/staging environments where AWS Secrets Manager is unavailable, use standard Kubernetes Secrets with:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backend-secrets
type: Opaque
stringData:
  DATABASE_URL: "postgres://..."
  RABBITMQ_PASS: "..."
```

**Note:** K8s Secrets are base64-encoded, not encrypted by default. For staging, enable **encryption at rest** via KMS.

---

## Rotation Procedures

### 1. Database Credentials Rotation (90-day cadence)

**Pre-requisites:** Admin access to PostgreSQL, write access to AWS Secrets Manager.

```bash
#!/bin/bash
# scripts/rotate-db-creds.sh
# Usage: ./rotate-db-creds.sh <environment> <new_password>

ENV=$1
NEW_PASS=$2
SECRET_ID="pudimproductivity/${ENV}/db-password"

# Step 1: Update the database user password
PGPASSWORD="$OLD_PASS" psql -h "$DB_HOST" -U pudim -d postgres \
  -c "ALTER USER pudim PASSWORD '$NEW_PASS';"

# Step 2: Update AWS Secrets Manager
aws secretsmanager put-secret-value \
  --secret-id "$SECRET_ID" \
  --secret-string "$NEW_PASS"

# Step 3: Trigger rolling restart of backend pods
kubectl rollout restart deployment/pudim-backend -n "$ENV"

# Step 4: Verify health
sleep 30
curl -f "https://${ENV}.api.pudimproductivity.com/api/v1/health" || {
  echo "ERROR: Health check failed after rotation"
  # Trigger rollback procedure
  exit 1
}

# Step 5: Verify old password is revoked (after all pods restarted)
# (If using createdb with password expiration, add here)

echo "Database credentials rotated successfully for ${ENV}"
```

**Rollback:** If health check fails, restore the old password via Secrets Manager and re-run the rollout.

### 2. API Key Rotation (30-day cadence)

| Key | Rotation Action |
|-----|----------------|
| FCM Server Key | Generate new key in Firebase Console → update Secrets Manager → restart notification worker |
| Google OAuth Client Secret | Generate new secret in Google Cloud Console → update Secrets Manager → restart backend |
| JWT Signing Key | Generate new key pair → update Secrets Manager → validate existing tokens still work → expire old key after 48h |

### 3. Certificate Rotation

- TLS certificates provisioned via **cert-manager** with Let's Encrypt (auto-renewal).
- Manual intervention only if DNS validation fails.

---

## Security Hardening Checklist

- [ ] `.env` is in `.gitignore` (already done)
- [ ] Database passwords are > 20 characters, mixed case, special chars
- [ ] Secrets Manager access logs enabled (CloudTrail)
- [ ] IAM roles follow least-privilege (secrets-reader policy allows only specific secrets)
- [ ] No hardcoded secrets in source code, Dockerfiles, or CI YAML
- [ ] Kubernetes Secrets encrypted at rest (KMS)
- [ ] Secrets Store CSI Driver configured to auto-rotate on secret update
- [ ] Dependency scanning includes secret detection (`gitleaks` or similar)
- [ ] Pre-commit hook to prevent accidental secret commits

---

## Incident Response: Secret Leak

1. **Immediately rotate** the leaked secret (see procedures above).
2. **Revoke access** for any leaked key/token.
3. **Audit logs** to determine scope of exposure (CloudTrail for AWS, DB logs for query history).
4. **Scan git history** for the leaked value:
   ```bash
   git log -p --all -S '<leaked-secret>'
   ```
5. **If found in git:**
   - Remove from git history with `git filter-branch` or `bfg repo-cleaner`.
   - Force-push cleaned history (notify all collaborators).
6. **Document** the incident in `docs/postmortems/` (see Phase 10).

---

## Local Development Security

### Pre-commit Hook

Install `gitleaks` for local secret scanning:

```bash
# From repo root
brew install gitleaks        # macOS
sudo apt install gitleaks    # Linux

# Install pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
gitleaks detect --source . --verbose
EOF
chmod +x .git/hooks/pre-commit
```

### Docker Compose Security

- Never expose PostgreSQL port `5432` directly from the host in production.
- Use the non-default port `5433` for local dev to avoid conflicts.
- RabbitMQ management UI should only be bound to `127.0.0.1` for local dev.