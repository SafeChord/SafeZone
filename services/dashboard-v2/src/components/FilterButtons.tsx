import type { Interval } from "@/types/api";

const INTERVALS: Interval[] = ["3", "7", "14", "30"];

interface Props {
  interval: Interval;
  ratio: boolean;
  onIntervalChange: (v: Interval) => void;
  onRatioChange: (v: boolean) => void;
}

export function FilterButtons({ interval, ratio, onIntervalChange, onRatioChange }: Props) {
  return (
    <div style={{ display: "flex", gap: "8px", alignItems: "center", flexWrap: "wrap" }}>
      {INTERVALS.map((d) => (
        <button
          key={d}
          onClick={() => onIntervalChange(d)}
          style={{
            padding: "6px 14px",
            borderRadius: "4px",
            border: "1px solid #ccc",
            background: interval === d ? "#2c7be5" : "#fff",
            color: interval === d ? "#fff" : "#333",
            cursor: "pointer",
            fontWeight: interval === d ? 600 : 400,
          }}
        >
          {d}d
        </button>
      ))}
      <label style={{ marginLeft: "12px", cursor: "pointer", userSelect: "none" }}>
        <input
          type="checkbox"
          checked={ratio}
          onChange={(e) => onRatioChange(e.target.checked)}
          style={{ marginRight: "4px" }}
        />
        Per capita
      </label>
    </div>
  );
}
