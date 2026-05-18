import { settings } from "@/config/settings";

export function generateTraceId(): string {
  return `dash-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export async function apiFetch<T>(path: string, base?: string): Promise<T> {
  const url = `${base ?? settings.apiBaseUrl}${path}`;
  const traceId = generateTraceId();

  const res = await fetch(url, {
    headers: { "X-Trace-ID": traceId },
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText} [${url}]`);
  }

  return res.json() as Promise<T>;
}
