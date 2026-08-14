import { type SortOption } from "../utils/sort";
import { useI18n } from "../i18n";

const SORT_LABEL_KEYS: Record<SortOption, string> = {
  "alpha-asc": "sort.alphaAsc",
  "alpha-desc": "sort.alphaDesc",
  "created-asc": "sort.createdAsc",
  "created-desc": "sort.createdDesc",
  "time-asc": "sort.timeAsc",
  "time-desc": "sort.timeDesc",
};

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
  const { t } = useI18n();
  return (
    <select
      className="sort-select"
      value={value}
      onChange={(e) => onChange(e.target.value as SortOption)}
      aria-label={t("sort.ariaLabel")}
    >
      {options.map((opt) => (
        <option key={opt} value={opt}>
          {t(SORT_LABEL_KEYS[opt])}
        </option>
      ))}
    </select>
  );
}