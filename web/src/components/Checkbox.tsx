interface CheckboxProps {
  checked: boolean;
  onChange: () => void;
  disabled?: boolean;
}

export default function Checkbox({
  checked,
  onChange,
  disabled = false,
}: CheckboxProps) {
  return (
    <label className="checkbox-wrapper" onClick={(e) => e.stopPropagation()}>
      <input
        type="checkbox"
        checked={checked}
        onChange={onChange}
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
