interface QuickAddFormProps {
  value: string;
  onChange: (v: string) => void;
  onSubmit: () => void;
  placeholder: string;
  submitLabel?: string;
  isPending?: boolean;
}

export default function QuickAddForm({
  value,
  onChange,
  onSubmit,
  placeholder,
  submitLabel = "Add",
  isPending = false,
}: QuickAddFormProps) {
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
        {isPending ? "..." : submitLabel}
      </button>
    </form>
  );
}