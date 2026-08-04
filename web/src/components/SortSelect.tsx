import { SORT_LABELS, type SortOption } from "../utils/sort";

interface SortSelectProps {
  value: SortOption;
  onChange: (option: SortOption) => void;
  options: SortOption[];
}

export default function SortSelect({
  value,
  onChange,
  options,
}: SortSelectProps) {
  return (
    <select
      className="sort-select"
      value={value}
      onChange={(e) => onChange(e.target.value as SortOption)}
      aria-label="Sort tasks"
    >
      {options.map((opt) => (
        <option key={opt} value={opt}>
          {SORT_LABELS[opt]}
        </option>
      ))}
    </select>
  );
}