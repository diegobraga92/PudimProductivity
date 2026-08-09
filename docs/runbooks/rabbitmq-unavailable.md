# Runbook — RabbitMQ Unavailable

Severity: **SEV-3** (notifications degraded; core app + sync unaffected)

## Symptoms
- No email/push notifications are delivered.
- Server logs show repeated: `composite bus: child publish failed (continuing)`
  or `rabbitmq connection lost`.
- `notifications.dlq` may grow (messages retried then discarded after 5×).

## Diagnosis
```bash
# Broker up?
docker ps | grep rabbitmq
curl -s -u pudim:CHANGEME http://localhost:15672/api/health/checks/alarms
# Backend connected?
grep -a 'rabbitmq' <backend log>
```

## Recovery
1. **Restart RabbitMQ** (`docker compose up -d rabbitmq`) or fix whatever
   took it down.
2. **Restart the backend** — the current adapter does not auto-reconnect after
   a connection loss (see postmortem 001, action item 1).
3. Verify: create a task → email arrives in Mailpit (`:8025`).

## Notes
- WebSocket sync is **not** affected (in-memory composite child).
- Events published while the broker was down are **lost** (by design; outbox
  pattern is the fix — postmortem 001, action item 3).

## Prevention
- RabbitMQ health check + Prometheus alert.
- Auto-reconnect in the adapter (tracked).
