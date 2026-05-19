import type { Interval } from "@/types/api";

const INTERVALS: { value: Interval; label: string }[] = [
  { value: "3", label: "Last 3 days" },
  { value: "7", label: "Last 7 days" },
  { value: "14", label: "Last 14 days" },
  { value: "30", label: "Last 30 days" },
];

interface Props {
  interval: Interval;
  ratio: boolean;
  onIntervalChange: (v: Interval) => void;
  onRatioChange: (v: boolean) => void;
}

const btnBase: React.CSSProperties = {
  padding: "6px 14px",
  border: "1px solid #ccc",
  cursor: "pointer",
  fontSize: "13px",
  background: "#fff",
  color: "#333",
};

export function FilterButtons({ interval, ratio, onIntervalChange, onRatioChange }: Props) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "8px" }}>
      {/* Left: interval buttons */}
      <div style={{ display: "flex" }}>
        {INTERVALS.map((d, i) => (
          <button
            key={d.value}
            onClick={() => onIntervalChange(d.value)}
            style={{
              ...btnBase,
              background: interval === d.value ? "#e9ecef" : "#fff",
              fontWeight: interval === d.value ? 600 : 400,
              borderRadius: i === 0 ? "4px 0 0 4px" : i === INTERVALS.length - 1 ? "0 4px 4px 0" : "0",
              borderLeft: i === 0 ? "1px solid #ccc" : "none",
            }}
          >
            {d.label}
          </button>
        ))}
      </div>

      {/* Right: cases/ratio toggle */}
      <div style={{ display: "flex" }}>
        <button
          onClick={() => onRatioChange(false)}
          style={{
            ...btnBase,
            borderRadius: "4px 0 0 4px",
            background: !ratio ? "#6c757d" : "#fff",
            color: !ratio ? "#fff" : "#333",
            fontWeight: !ratio ? 600 : 400,
          }}
        >
          Cases
        </button>
        <button
          onClick={() => onRatioChange(true)}
          style={{
            ...btnBase,
            borderRadius: "0 4px 4px 0",
            borderLeft: "none",
            background: ratio ? "#6c757d" : "#fff",
            color: ratio ? "#fff" : "#333",
            fontWeight: ratio ? 600 : 400,
          }}
        >
          Ratio
        </button>
      </div>
    </div>
  );
}
