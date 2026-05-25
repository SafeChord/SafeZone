import { useState, useEffect, useCallback } from "react";
import {
  fetchNationalCases,
  fetchCityCases,
} from "@/services/caseService";
import type { AnalyticsAPIData, Interval } from "@/types/api";

export interface CityRow {
  city: string;
  cases: number;
  ratio?: number;
}

export function useCases(systemDate: string | null, interval: Interval, ratio: boolean) {
  const [national, setNational] = useState<AnalyticsAPIData | null>(null);
  const [cities, setCities] = useState<CityRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const cityNames = [
    "台北市", "新北市", "桃園市", "台中市", "台南市", "高雄市",
    "基隆市", "新竹市", "嘉義市", "新竹縣", "苗栗縣", "彰化縣",
    "南投縣", "雲林縣", "嘉義縣", "屏東縣", "宜蘭縣", "花蓮縣",
    "台東縣", "澎湖縣", "金門縣", "連江縣",
  ];

  const load = useCallback(async () => {
    if (!systemDate) return;
    setLoading(true);
    setError(null);

    try {
      const [natRes, ...cityResults] = await Promise.all([
        fetchNationalCases(systemDate, interval),
        ...cityNames.map((city) => fetchCityCases(systemDate, interval, city, ratio)),
      ]);

      if (natRes.data) setNational(natRes.data);

      const rows: CityRow[] = cityResults
        .filter((r) => r.success && r.data)
        .map((r) => ({
          city: r.data!.city!,
          cases: r.data!.aggregated_cases ?? 0,
          ratio: r.data!.cases_population_ratio,
        }));

      setCities(rows);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [systemDate, interval, ratio]);

  useEffect(() => {
    load();
  }, [load]);

  return { national, cities, loading, error };
}
