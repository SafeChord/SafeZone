import { useState, useEffect } from "react";
import { fetchNationalCases } from "@/services/caseService";

interface Props {
  systemDate: string | null;
}

export function StatCards({ systemDate }: Props) {
  const [weekCases, setWeekCases] = useState<number | null>(null);
  const [todayCases, setTodayCases] = useState<number | null>(null);

  useEffect(() => {
    if (!systemDate) return;

    Promise.all([
      fetchNationalCases(systemDate, "7"),
      fetchNationalCases(systemDate, "1"),
    ]).then(([weekRes, todayRes]) => {
      setWeekCases(weekRes.data?.aggregated_cases ?? null);
      setTodayCases(todayRes.data?.aggregated_cases ?? null);
    });
  }, [systemDate]);

  return (
    <div
      style={{
        background: "#2c3e50",
        borderRadius: "8px",
        padding: "20px",
        color: "#fff",
      }}
    >
      <div style={{ fontSize: "16px", fontWeight: 700, marginBottom: "8px" }}>
        近七日新增案例
      </div>
      <div
        style={{
          background: "#2ecc71",
          borderRadius: "4px",
          padding: "8px 16px",
          fontSize: "28px",
          fontWeight: 700,
          textAlign: "right",
          marginBottom: "12px",
        }}
      >
        {weekCases?.toLocaleString() ?? "-"}
      </div>
      <div style={{ fontSize: "16px", fontWeight: 700, marginBottom: "8px" }}>
        今日新增案例
      </div>
      <div
        style={{
          background: "#2ecc71",
          borderRadius: "4px",
          padding: "8px 16px",
          fontSize: "28px",
          fontWeight: 700,
          textAlign: "right",
        }}
      >
        {todayCases?.toLocaleString() ?? "-"}
      </div>
    </div>
  );
}
