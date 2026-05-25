import { apiFetch } from "./apiClient";
import { settings } from "@/config/settings";
import type {
  AnalyticsAPIResponse,
  Interval,
} from "@/types/api";

function toQuery(params: Record<string, string | boolean | undefined>): string {
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== false,
  );
  if (entries.length === 0) return "";
  return "?" + entries.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join("&");
}

export async function fetchNationalCases(
  now: string,
  interval: Interval,
): Promise<AnalyticsAPIResponse> {
  return apiFetch<AnalyticsAPIResponse>(
    `/cases/national${toQuery({ now, interval })}`,
    settings.apiBaseUrl,
  );
}

export async function fetchCityCases(
  now: string,
  interval: Interval,
  city: string,
  ratio?: boolean,
): Promise<AnalyticsAPIResponse> {
  return apiFetch<AnalyticsAPIResponse>(
    `/cases/city${toQuery({ now, interval, city, ratio })}`,
    settings.apiBaseUrl,
  );
}

export async function fetchRegionCases(
  now: string,
  interval: Interval,
  city: string,
  region: string,
  ratio?: boolean,
): Promise<AnalyticsAPIResponse> {
  return apiFetch<AnalyticsAPIResponse>(
    `/cases/region${toQuery({ now, interval, city, region, ratio })}`,
    settings.apiBaseUrl,
  );
}
