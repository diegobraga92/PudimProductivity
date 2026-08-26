import { useState, useEffect } from "react";
import { createTask, parseTask, type RecurrenceDay } from "../api/tasks";
import ScheduleFields from "../components/ScheduleFields";
import RecurrenceDayPicker from "../components/RecurrenceDayPicker";
import Modal from "../components/Modal";
import { useI18n } from "../i18n";
import { COLOR_PALETTE } from "../utils/constants";

interface TaskCreateProps {
  onCreated: () => void;
  onCancel: () => void;
}

export default function TaskCreate({ onCreated, onCancel }: TaskCreateProps) {
  const { t } = useI18n();
  const [title, setTitle] = useState("");
  const [isHabit, setIsHabit] = useState(false);
  const [selectedDays, setSelectedDays] = useState<RecurrenceDay[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Scheduling fields
  const [showSchedule, setShowSchedule] = useState(false);
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("10:00");
  const [color, setColor] = useState(COLOR_PALETTE[0]);
  const [scheduledDate, setScheduledDate] = useState("");
  const [alarmMinutes, setAlarmMinutes] = useState("");

  // Which field a validation error belongs to, so it can be highlighted.
  const [errorField, setErrorField] = useState<"title" | "days" | "schedule" | null>(null);

  // Check for planner prefill data
  useEffect(() => {
    const prefillJson = sessionStorage.getItem("planner_prefill");
    if (prefillJson) {
      try {
        const prefill = JSON.parse(prefillJson);
        setShowSchedule(true);
        setStartTime(prefill.start_time || "09:00");
        setEndTime(prefill.end_time || "10:00");
        setColor(COLOR_PALETTE[0]);

        // Map the day to the scheduled date for one-off tasks
        // For habits, we set the day
        if (prefill.day) {
          setSelectedDays([prefill.day as RecurrenceDay]);
        }

        sessionStorage.removeItem("planner_prefill");
      } catch {
        // Ignore parse errors
      }
    }
  }, []);

  const toggleDay = (day: RecurrenceDay) => {
    setSelectedDays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
    );
  };

  // Smart Parse: natural-language input → pre-filled form.
  const [showParse, setShowParse] = useState(false);
  const [parseInput, setParseInput] = useState("");
  const [parsing, setParsing] = useState(false);
  const [parseError, setParseError] = useState<string | null>(null);

  const handleParse = async () => {
    if (!parseInput.trim()) return;
    setParsing(true);
    setParseError(null);
    try {
      const result = await parseTask(parseInput.trim());
      if (result.title) setTitle(result.title);
      if (result.due_date) {
        setShowSchedule(true);
        setScheduledDate(result.due_date);
      }
      if (result.start_time) {
        setShowSchedule(true);
        setStartTime(result.start_time);
      }
      if (result.end_time) setEndTime(result.end_time);
      if (result.recurrence_days && result.recurrence_days.length > 0) {
        setIsHabit(true);
        setSelectedDays(result.recurrence_days);
      }
      setParseInput("");
      setShowParse(false);
    } catch (err) {
      setParseError((err as Error).message);
    } finally {
      setParsing(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setErrorField(null);

    if (!title.trim()) {
      setError(t("tasks.validation.titleRequired"));
      setErrorField("title");
      return;
    }

    if (isHabit && selectedDays.length === 0) {
      setError(t("tasks.validation.dayRequired"));
      setErrorField("days");
      return;
    }

    if (showSchedule && !isHabit && !scheduledDate) {
      setError(t("tasks.validation.dateRequired"));
      setErrorField("schedule");
      return;
    }

    if (showSchedule && startTime >= endTime) {
      setError(t("tasks.validation.timeOrder"));
      setErrorField("schedule");
      return;
    }

    setSubmitting(true);
    try {
      await createTask({
        title: title.trim(),
        recurrence_days: isHabit ? selectedDays : undefined,
        start_time: showSchedule ? startTime : undefined,
        end_time: showSchedule ? endTime : undefined,
        color: showSchedule ? color : undefined,
        scheduled_date: showSchedule && !isHabit ? scheduledDate : undefined,
        alarm_minutes: showSchedule && isHabit && alarmMinutes !== "" ? Number(alarmMinutes) : undefined,
      });
      onCreated();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal onClose={onCancel} maxWidth={480}>
      <div className="flex-between" style={{ marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>
          {t("tasks.newTaskTitle")}
        </h2>
        <button className="btn btn-ghost btn-sm" onClick={onCancel} aria-label={t("a11y.close")}>
          ✕
        </button>
      </div>

      <form onSubmit={handleSubmit}>
        {/* Smart Parse: quick natural-language entry */}
        <div className="card" style={{ marginBottom: "var(--space-lg)" }}>
          {!showParse ? (
            <button
              type="button"
              className="btn btn-ghost"
              style={{ width: "100%", textAlign: "center" }}
              onClick={() => setShowParse(true)}
            >
              {t("tasks.smartParse")}
            </button>
          ) : (
            <div>
              <label className="form-label">{t("tasks.describeTask")}</label>
              <input
                type="text"
                className="input"
                value={parseInput}
                onChange={(e) => setParseInput(e.target.value)}
                placeholder={t("tasks.parsePlaceholder")}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleParse();
                  }
                }}
              />
              <div style={{ display: "flex", gap: "var(--space-sm)", marginTop: "var(--space-sm)" }}>
                <button type="button" className="btn btn-primary" disabled={parsing} onClick={handleParse}>
                  {parsing ? t("tasks.parsing") : t("tasks.parse")}
                </button>
                <button type="button" className="btn btn-ghost" onClick={() => setShowParse(false)}>
                  {t("common.cancel")}
                </button>
              </div>
              {parseError && <p style={{ color: "var(--color-danger)", fontSize: "var(--font-size-sm)" }}>{parseError}</p>}
            </div>
          )}
        </div>

        <div className="card" style={{ marginBottom: "var(--space-lg)" }}>
          <div style={{ marginBottom: "var(--space-md)" }}>
            <label className="form-label">
              {t("tasks.whatToDo")}
            </label>
            <input
              type="text"
              className={`input ${errorField === "title" ? "input--error" : ""}`}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("tasks.titlePlaceholder")}
              aria-invalid={errorField === "title"}
              autoFocus
            />
          </div>

          {/* Habit toggle */}
          <div className="toggle-box">
            <label className="toggle-label">
              <input
                type="checkbox"
                checked={isHabit}
                onChange={(e) => {
                  setIsHabit(e.target.checked);
                  if (!e.target.checked) setSelectedDays([]);
                }}
                className="toggle-checkbox"
                style={{ accentColor: "var(--color-habit)" }}
              />
              {t("tasks.makeHabit")}
            </label>
          </div>

          {/* Day picker */}
          {isHabit && (
            <div style={{ marginBottom: "var(--space-md)" }}>
              <label className="form-label">
                {t("tasks.repeatOn")}
              </label>
              <RecurrenceDayPicker
                selectedDays={selectedDays}
                onToggle={toggleDay}
                error={errorField === "days"}
              />
            </div>
          )}

          <ScheduleFields
            showSchedule={showSchedule}
            onToggleSchedule={setShowSchedule}
            startTime={startTime}
            endTime={endTime}
            color={color}
            scheduledDate={scheduledDate}
            alarmMinutes={alarmMinutes}
            isHabit={isHabit}
            onStartTimeChange={setStartTime}
            onEndTimeChange={setEndTime}
            onColorChange={setColor}
            onScheduledDateChange={setScheduledDate}
            onAlarmMinutesChange={setAlarmMinutes}
          />

          {error && (
            <div className="form-error-banner" role="alert">
              {error}
            </div>
          )}
        </div>

        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={submitting}
          >
            {submitting ? t("tasks.adding") : t("tasks.addTask")}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={onCancel}
          >
            {t("common.cancel")}
          </button>
        </div>
      </form>
    </Modal>
  );
}