import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import type { CityRow } from "@/hooks/useCases";

interface Props {
  cities: CityRow[];
  ratio: boolean;
}

export function TopCitiesChart({ cities, ratio }: Props) {
  const sorted = [...cities]
    .sort((a, b) => (ratio ? (b.ratio ?? 0) - (a.ratio ?? 0) : b.cases - a.cases))
    .slice(0, 10);

  const dataKey = ratio ? "ratio" : "cases";
  const label = ratio ? "Cases / Population (%)" : "Cases";

  return (
    <div style={{ background: "#fff", borderRadius: "8px", border: "1px solid #e0e0e0", padding: "16px" }}>
      <h3 style={{ margin: "0 0 12px", fontSize: "14px", color: "#555" }}>
        Top 10 Cities — {label}
      </h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={sorted} layout="vertical" margin={{ left: 60 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" />
          <YAxis type="category" dataKey="city" width={60} tick={{ fontSize: 12 }} />
          <Tooltip />
          <Bar dataKey={dataKey} fill="#2c7be5" radius={[0, 4, 4, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
