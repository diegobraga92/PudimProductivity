interface CheckboxProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  title?: string;
}

export default function Checkbox({
  checked,
  onChange,
  disabled = false,
  title,
}: CheckboxProps) {
  return (
    <label className="checkbox-wrapper" title={title} onClick={(e) => e.stopPropagation()}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        disabled={disabled}
      />
      <span className="checkbox-custom">
        <svg viewBox="0 0 24 24">
          <polyline points="20 6 9 17 4 12" />
        </svg>
      </span>
    </label>
  );
}
