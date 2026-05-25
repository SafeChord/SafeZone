import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiFetch, generateTraceId } from "@/services/apiClient";

// Mock settings
vi.mock("@/config/settings", () => ({
  settings: {
    apiBaseUrl: "http://test-api",
    timeBaseUrl: "http://test-time",
    defaultInterval: 7,
    pollIntervalMs: 60000,
  },
}));

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

beforeEach(() => {
  vi.clearAllMocks();
});

describe("generateTraceId", () => {
  it("returns a string starting with 'dash-'", () => {
    const id = generateTraceId();
    expect(id).toMatch(/^dash-\d+-[a-z0-9]+$/);
  });

  it("generates unique IDs", () => {
    const ids = new Set(Array.from({ length: 100 }, () => generateTraceId()));
    expect(ids.size).toBe(100);
  });
});

describe("apiFetch", () => {
  it("sends GET with X-Trace-ID header", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ success: true }),
    });

    await apiFetch("/test");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, init] = mockFetch.mock.calls[0]!;
    expect(url).toBe("http://test-api/test");
    expect(init.headers["X-Trace-ID"]).toMatch(/^dash-/);
  });

  it("uses custom base URL when provided", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: 1 }),
    });

    await apiFetch("/now", "http://custom");

    const [url] = mockFetch.mock.calls[0]!;
    expect(url).toBe("http://custom/now");
  });

  it("throws on non-OK response", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
    });

    await expect(apiFetch("/fail")).rejects.toThrow("API error: 500");
  });

  it("returns parsed JSON on success", async () => {
    const payload = { success: true, data: { count: 42 } };
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(payload),
    });

    const result = await apiFetch("/data");
    expect(result).toEqual(payload);
  });
});
