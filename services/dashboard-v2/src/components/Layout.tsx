import { useState } from "react";
import type { Interval } from "@/types/api";
import { useSystemTime } from "@/hooks/useSystemTime";
import { useCases } from "@/hooks/useCases";
import { RiskMap } from "./RiskMap";
import { StatCards } from "./StatCards";
import { TopCitiesChart } from "./TopCitiesChart";
import { FilterButtons } from "./FilterButtons";

export function Layout() {
  const [interval, setInterval] = useState<Interval>("7");
  const [ratio, setRatio] = useState(false);

  const { systemDate, error: timeError } = useSystemTime();
  const { national, cities, loading, error: caseError } = useCases(systemDate, interval, ratio);

  const error = timeError ?? caseError;

  return (
    <div style={{ fontFamily: "system-ui, sans-serif", background: "#f5f7fa", minHeight: "100vh" }}>
      <header style={{ background: "#2c7be5", color: "#fff", padding: "12px 24px" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h1 style={{ margin: 0, fontSize: "18px", fontWeight: 600 }}>SafeZone Dashboard</h1>
          <span style={{ fontSize: "13px", opacity: 0.85 }}>
            {systemDate ? `System Date: ${systemDate}` : "Loading time..."}
          </span>
        </div>
      </header>

      <main style={{ padding: "20px 24px", maxWidth: "1400px", margin: "0 auto" }}>
        {error && (
          <div style={{ background: "#ffeaea", border: "1px solid #e57373", borderRadius: "4px", padding: "10px 16px", marginBottom: "16px", color: "#c62828" }}>
            {error}
          </div>
        )}

        <div style={{ marginBottom: "16px" }}>
          <FilterButtons
            interval={interval}
            ratio={ratio}
            onIntervalChange={setInterval}
            onRatioChange={setRatio}
          />
        </div>

        <StatCards national={national} loading={loading} />

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "2fr 1fr",
            gap: "20px",
            marginTop: "20px",
          }}
        >
          <div style={{ background: "#fff", borderRadius: "8px", border: "1px solid #e0e0e0", overflow: "hidden" }}>
            <RiskMap cities={cities} ratio={ratio} />
          </div>
          <div>
            <TopCitiesChart cities={cities} ratio={ratio} />
          </div>
        </div>
      </main>
    </div>
  );
}
