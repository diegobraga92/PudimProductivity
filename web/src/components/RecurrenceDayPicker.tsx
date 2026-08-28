import { DAY_OPTIONS } from "../utils/constants";
import type { RecurrenceDay } from "../api/tasks";
import { useI18n } from "../i18n";

interface RecurrenceDayPickerProps {
  selectedDays: RecurrenceDay[];
  onToggle: (day: RecurrenceDay) => void;
  /** Highlight the buttons as invalid (e.g. a habit with no days selected). */
  error?: boolean;
}

/**
 * Weekday toggle buttons for habit recurrence. Shared by the create and edit
 * task modals (extracted from TaskCreate).
 */
export default function RecurrenceDayPicker({
  selectedDays,
  onToggle,
  error = false,
}: RecurrenceDayPickerProps) {
  const { t } = useI18n();
  return (
    <div style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
      {DAY_OPTIONS.map(({ value }) => {
        const isSelected = selectedDays.includes(value);
        return (
          <button
            key={value}
            type="button"
            onClick={() => onToggle(value)}
            style={{
              padding: "0.4rem 0.8rem",
              border: isSelected
                ? "2px solid var(--color-habit)"
                : error
                ? "1.5px solid var(--color-danger)"
                : "1.5px solid var(--color-border)",
              borderRadius: "var(--radius-sm)",
              background: isSelected ? "var(--color-habit-light)" : "var(--color-surface)",
              cursor: "pointer",
              fontWeight: isSelected ? 600 : 400,
              color: isSelected ? "var(--color-habit)" : "var(--color-text-secondary)",
              fontFamily: "var(--font-family)",
              fontSize: "var(--font-size-sm)",
              transition: "all var(--transition-fast)",
            }}
          >
            {t(`days.${value}`)}
          </button>
        );
      })}
    </div>
  );
}
