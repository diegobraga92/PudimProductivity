import { useQuery } from "@tanstack/react-query";
import { getDailySchedule, type ScheduleSlot } from "../api/scheduler";

function timeLabel(hhmm: string): string {
  const [h, m] = hhmm.split(":").map(Number);
  const suffix = h >= 12 ? "pm" : "am";
  const hour = ((h + 11) % 12) + 1;
  return `${hour}:${String(m).padStart(2, "0")} ${suffix}`;
}

export default function DailyPlan() {
  const { data: plan, isLoading, error } = useQuery({
    queryKey: ["schedule"],
    queryFn: () => getDailySchedule(),
  });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>🗓 Daily Plan</h2>
        {plan && (
          <span className="badge badge-habit">
            {plan.avg_per_day.toFixed(1)}/day avg · {plan.free_hours}h free
          </span>
        )}
      </div>

      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>Planning your day…</p>}
      {error && <p style={{ color: "var(--color-danger)" }}>{(error as Error).message}</p>}

      {plan && plan.slots.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">🗓</div>
          <p className="empty-state-text">
            Nothing to schedule — no pending tasks or habits for today.
          </p>
        </div>
      )}

      {plan && plan.slots.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-sm)" }}>
          {plan.slots.map((slot: ScheduleSlot, i) => (
            <div key={`${slot.task_id}-${i}`} className="card" style={{ display: "flex", alignItems: "center", gap: "var(--space-md)" }}>
              <div style={{ minWidth: 110, textAlign: "right", fontVariantNumeric: "tabular-nums" }}>
                <div style={{ fontWeight: 600 }}>{timeLabel(slot.start_time)}</div>
                <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
                  {timeLabel(slot.end_time)}
                </div>
              </div>
              <div style={{ width: 3, alignSelf: "stretch", background: slot.kind === "habit" ? "var(--color-habit)" : "var(--color-primary)" }} />
              <div style={{ flex: 1 }}>
                <span className="card-title" style={{ fontSize: "var(--font-size-md)" }}>{slot.title}</span>
                <span className={`badge ${slot.kind === "habit" ? "badge-habit" : "badge-todo"}`} style={{ marginLeft: "var(--space-sm)" }}>
                  {slot.kind}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
