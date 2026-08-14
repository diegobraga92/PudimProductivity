interface QuickAddFormProps {
  value: string;
  onChange: (v: string) => void;
  onSubmit: () => void;
  placeholder: string;
  submitLabel?: string;
  isPending?: boolean;
}

import { useI18n } from "../i18n";

export default function QuickAddForm({
  value,
  onChange,
  onSubmit,
  placeholder,
  submitLabel,
  isPending = false,
}: QuickAddFormProps) {
  const { t } = useI18n();
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
      style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-md)" }}
    >
      <input
        type="text"
        className="input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
      />
      <button
        type="submit"
        className="btn btn-primary"
        disabled={isPending || !value.trim()}
      >
        {isPending ? "..." : (submitLabel ?? t("common.add"))}
      </button>
    </form>
  );
}