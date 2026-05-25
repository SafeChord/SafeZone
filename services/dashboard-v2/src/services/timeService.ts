import { apiFetch } from "./apiClient";
import { settings } from "@/config/settings";
import type { SystemDateResponse } from "@/types/api";

export async function fetchSystemDate(): Promise<string> {
  const res = await apiFetch<SystemDateResponse>("/now", settings.timeBaseUrl);
  if (!res.success || !res.system_date) {
    throw new Error(`Time server error: ${res.message}`);
  }
  return res.system_date;
}
