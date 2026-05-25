import { describe, it, expect, vi, beforeEach } from "vitest";
import type { AnalyticsAPIResponse } from "@/types/api";

// Mock apiClient
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

// Import after mocks
const { fetchNationalCases, fetchCityCases, fetchRegionCases } = await import(
  "@/services/caseService"
);

const mockResponse: AnalyticsAPIResponse = {
  success: true,
  message: "OK",
  timestamp: "2026-01-01T00:00:00Z",
  data: {
    start_date: "2026-01-01",
    end_date: "2026-01-07",
    aggregated_cases: 100,
  },
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("fetchNationalCases", () => {
  it("calls apiFetch with correct path and query", async () => {
    mockApiFetch.mockResolvedValueOnce(mockResponse);

    const result = await fetchNationalCases("2026-01-07", "7");

    expect(mockApiFetch).toHaveBeenCalledWith(
      "/cases/national?now=2026-01-07&interval=7",
      "http://test-api",
    );
    expect(result).toEqual(mockResponse);
  });
});

describe("fetchCityCases", () => {
  it("includes city and excludes ratio when false", async () => {
    mockApiFetch.mockResolvedValueOnce(mockResponse);

    await fetchCityCases("2026-01-07", "7", "Taipei");

    const [path] = mockApiFetch.mock.calls[0]!;
    expect(path).toContain("city=Taipei");
    expect(path).not.toContain("ratio");
  });

  it("includes ratio=true when set", async () => {
    mockApiFetch.mockResolvedValueOnce(mockResponse);

    await fetchCityCases("2026-01-07", "7", "Taipei", true);

    const [path] = mockApiFetch.mock.calls[0]!;
    expect(path).toContain("ratio=true");
  });
});

describe("fetchRegionCases", () => {
  it("includes city and region in query", async () => {
    mockApiFetch.mockResolvedValueOnce(mockResponse);

    await fetchRegionCases("2026-01-07", "7", "Taipei", "Xinyi");

    const [path] = mockApiFetch.mock.calls[0]!;
    expect(path).toContain("city=Taipei");
    expect(path).toContain("region=Xinyi");
  });
});
