import { useState, useEffect, useCallback } from "react";
import { fetchCityCases } from "@/services/caseService";
import type { CityRow } from "./useCases";

const CITY_NAMES = [
  "台北市", "新北市", "桃園市", "台中市", "台南市", "高雄市",
  "基隆市", "新竹市", "嘉義市", "新竹縣", "苗栗縣", "彰化縣",
  "南投縣", "雲林縣", "嘉義縣", "屏東縣", "宜蘭縣", "花蓮縣",
  "台東縣", "澎湖縣", "金門縣", "連江縣",
];

/** Always fetches 7-day city data, independent of the interval selector. */
export function useTopCities(systemDate: string | null, ratio: boolean) {
  const [cities, setCities] = useState<CityRow[]>([]);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!systemDate) return;
    setError(null);

    try {
      const results = await Promise.all(
        CITY_NAMES.map((city) => fetchCityCases(systemDate, "7", city, ratio)),
      );

      const rows: CityRow[] = results
        .filter((r) => r.success && r.data)
        .map((r) => ({
          city: r.data!.city!,
          cases: r.data!.aggregated_cases ?? 0,
          ratio: r.data!.cases_population_ratio,
        }));

      setCities(rows);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  }, [systemDate, ratio]);

  useEffect(() => {
    load();
  }, [load]);

  return { topCities: cities, topCitiesError: error };
}
