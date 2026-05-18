import { describe, it, expect, vi, beforeEach } from "vitest";
import type { SystemDateResponse } from "@/types/api";

const mockApiFetch = vi.fn();
vi.mock("@/services/apiClient", () => ({
  apiFetch: (...args: unknown[]) => mockApiFetch(...args),
}));

vi.mock("@/config/settings", () => ({
  settings: {
    apiBaseUrl: "http://test-api",
    timeBaseUrl: "http://test-time",
    defaultInterval: 7,
    pollIntervalMs: 60000,
  },
}));

const { fetchSystemDate } = await import("@/services/timeService");

beforeEach(() => {
  vi.clearAllMocks();
});

describe("fetchSystemDate", () => {
  it("returns system_date on success", async () => {
    const res: SystemDateResponse = {
      success: true,
      message: "OK",
      timestamp: "2026-01-01T00:00:00Z",
      system_date: "2026-01-07",
    };
    mockApiFetch.mockResolvedValueOnce(res);

    const date = await fetchSystemDate();

    expect(date).toBe("2026-01-07");
    expect(mockApiFetch).toHaveBeenCalledWith("/now", "http://test-time");
  });

  it("throws when success is false", async () => {
    const res: SystemDateResponse = {
      success: false,
      message: "Redis down",
      timestamp: "2026-01-01T00:00:00Z",
    };
    mockApiFetch.mockResolvedValueOnce(res);

    await expect(fetchSystemDate()).rejects.toThrow("Time server error: Redis down");
  });

  it("throws when system_date is missing", async () => {
    const res: SystemDateResponse = {
      success: true,
      message: "OK",
      timestamp: "2026-01-01T00:00:00Z",
    };
    mockApiFetch.mockResolvedValueOnce(res);

    await expect(fetchSystemDate()).rejects.toThrow("Time server error");
  });
});
