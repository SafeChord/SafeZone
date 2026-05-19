import type { CityRow } from "@/hooks/useCases";

interface Props {
  cities: CityRow[];
  ratio: boolean;
  systemDate: string | null;
}

export function TopCitiesChart({ cities, ratio, systemDate }: Props) {
  const sorted = [...cities]
    .sort((a, b) => (ratio ? (b.ratio ?? 0) - (a.ratio ?? 0) : b.cases - a.cases))
    .slice(0, 10);

  const maxVal = sorted.length > 0
    ? Math.max(...sorted.map((r) => (ratio ? (r.ratio ?? 0) : r.cases)), 1)
    : 1;

  const label = ratio ? "Ratio" : "Cases";

  // Compute date range for subtitle (7-day window)
  let subtitle = "";
  if (systemDate) {
    const end = new Date(systemDate);
    const start = new Date(end);
    start.setDate(start.getDate() - 6);
    const fmt = (d: Date) => d.toISOString().slice(0, 10);
    subtitle = `Statistics: ${fmt(start)} ~ ${fmt(end)}`;
  }

  return (
    <div style={{ background: "#fff", borderRadius: "8px", border: "1px solid #e0e0e0", padding: "16px" }}>
      <h3 style={{ margin: "0 0 4px", fontSize: "15px", color: "#2c7be5" }}>
        Top 10 Cities
      </h3>
      {subtitle && (
        <p style={{ margin: "0 0 12px", fontSize: "12px", color: "#888" }}>{subtitle}</p>
      )}

      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "13px" }}>
        <thead>
          <tr style={{ borderBottom: "1px solid #eee" }}>
            <th style={{ textAlign: "left", padding: "4px 0", fontWeight: 500, color: "#888" }}>City</th>
            <th style={{ textAlign: "right", padding: "4px 8px", fontWeight: 500, color: "#888", width: "50px" }}>{label}</th>
            <th style={{ padding: "4px 0" }}></th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((row, i) => {
            const val = ratio ? (row.ratio ?? 0) : row.cases;
            const pct = (val / maxVal) * 100;
            const opacity = 1 - i * 0.07;
            return (
              <tr key={row.city} style={{ borderBottom: "1px solid #f5f5f5" }}>
                <td style={{ padding: "5px 0", whiteSpace: "nowrap" }}>{row.city}</td>
                <td style={{ textAlign: "right", padding: "5px 8px", fontWeight: 500 }}>
                  {ratio ? val.toFixed(2) : val.toLocaleString()}
                </td>
                <td style={{ padding: "5px 0", width: "50%" }}>
                  <div
                    style={{
                      height: "16px",
                      width: `${pct}%`,
                      background: `rgba(44, 123, 229, ${opacity})`,
                      borderRadius: "2px",
                      minWidth: "2px",
                    }}
                  />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
