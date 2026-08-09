# Runbook — WebSocket Disconnect Storm

Severity: **SEV-3** (client churn; REST load spike)

## Symptoms
- Log spam: repeated `ws client connected` / `ws client disconnected`.
- REST request rate spikes (clients fall back to `stale` → full refetch).
- No errors on the API itself.

## Typical Causes
- Network blip / mobile app backgrounding churn.
- Backend restart (all sockets drop at once → all clients reconnect together).
- Sync hub hitting `MaxClients` (default 0 = unlimited; if set, excess
  connections get 1006 Policy Violation).

## Diagnosis
```bash
grep -ac 'ws client connected' <backend log>
curl -s http://localhost:9090/metrics | grep http_requests_in_flight
# Client reconnect pattern (should be exponential backoff, not simultaneous)
grep -a 'ws client connected' <backend log> | tail -50 | uniq -c | head
```

## Recovery
1. Usually self-healing: clients reconnect with exponential backoff (1s → 30s)
   and replay from `last_seq`; a `stale` event triggers one REST refetch each.
2. If the storm is caused by a backend restart, the reconnect burst is a known
   thundering-herd — it settles within ~30s.
3. If clients are stuck reconnecting (loop), check `wsURL` config (base URL /
   proxy `Upgrade` headers) and the sync hub `MaxClients`.

## Prevention
- Client exponential backoff (already implemented, web + mobile).
- If bursty reconnects become a problem: jitter the backoff, or scale the sync
  hub horizontally (Phase 8 / presence).
