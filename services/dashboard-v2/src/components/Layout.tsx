import { useState, useMemo } from "react";
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
  const { cities, error: caseError } = useCases(systemDate, interval, ratio);

  const error = timeError ?? caseError;

  const bannerText = useMemo(() => {
    if (!systemDate) return "";
    const end = new Date(systemDate);
    const start = new Date(end);
    start.setDate(start.getDate() - (Number(interval) - 1));
    const fmt = (d: Date) => d.toISOString().slice(0, 10);
    const mode = ratio ? "Ratio" : "Cases";
    return `${fmt(start)} ~ ${fmt(end)} Risk Map — by ${mode}`;
  }, [systemDate, interval, ratio]);

  return (
    <div style={{ fontFamily: "system-ui, sans-serif", background: "#f5f7fa", minHeight: "100vh" }}>
      <header style={{ background: "#2c3e50", color: "#fff", padding: "12px 24px" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h1 style={{ margin: 0, fontSize: "20px", fontWeight: 600 }}>SafeZone Dashboard</h1>
          <span style={{ fontSize: "13px", background: "rgba(255,255,255,0.15)", padding: "4px 10px", borderRadius: "4px" }}>
            {systemDate ? `⏱ Simulated: ${systemDate}` : "Loading time..."}
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

        {/* 8:4 grid — map left, cards right */}
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "2fr 1fr",
            gap: "20px",
          }}
        >
          {/* Left: banner + map */}
          <div>
            <div
              style={{
                background: "#2c7be5",
                color: "#fff",
                padding: "8px 16px",
                fontSize: "14px",
                fontWeight: 500,
                borderRadius: "8px 8px 0 0",
              }}
            >
              {bannerText || "Loading..."}
            </div>
            <div style={{ background: "#fff", border: "1px solid #e0e0e0", borderTop: "none", borderRadius: "0 0 8px 8px", overflow: "hidden" }}>
              <RiskMap
                cities={cities}
                ratio={ratio}
                systemDate={systemDate}
                interval={interval}
              />
            </div>
          </div>

          {/* Right: cases card + top cities stacked */}
          <div style={{ display: "flex", flexDirection: "column", gap: "20px" }}>
            <StatCards systemDate={systemDate} />
            <TopCitiesChart cities={cities} ratio={ratio} systemDate={systemDate} />
          </div>
        </div>
      </main>
    </div>
  );
}
