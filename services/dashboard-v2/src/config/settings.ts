export const settings = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "/dashboard/api",
  timeBaseUrl: import.meta.env.VITE_TIME_BASE_URL ?? "/dashboard/time",
  defaultInterval: 7,
  pollIntervalMs: 60_000,
} as const;
