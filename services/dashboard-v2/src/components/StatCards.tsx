import type { AnalyticsAPIData } from "@/types/api";

interface Props {
  national: AnalyticsAPIData | null;
  loading: boolean;
}

export function StatCards({ national, loading }: Props) {
  if (loading) {
    return <div style={{ padding: "16px", color: "#888" }}>Loading...</div>;
  }

  return (
    <div style={{ display: "flex", gap: "16px", flexWrap: "wrap" }}>
      <Card
        label="Total Cases"
        value={national?.aggregated_cases?.toLocaleString() ?? "-"}
      />
      <Card
        label="Period"
        value={
          national ? `${national.start_date} ~ ${national.end_date}` : "-"
        }
      />
    </div>
  );
}

function Card({ label, value }: { label: string; value: string }) {
  return (
    <div
      style={{
        background: "#fff",
        border: "1px solid #e0e0e0",
        borderRadius: "8px",
        padding: "16px 24px",
        minWidth: "180px",
      }}
    >
      <div style={{ fontSize: "12px", color: "#888", marginBottom: "4px" }}>
        {label}
      </div>
      <div style={{ fontSize: "22px", fontWeight: 700 }}>{value}</div>
    </div>
  );
}
