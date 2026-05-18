// Mirror of utils/pydantic_model/response.py

export interface APIResponse {
  success: boolean;
  message: string;
  detail?: string;
  timestamp: string;
}

export interface AnalyticsAPIData {
  start_date: string;
  end_date: string;
  city?: string;
  region?: string;
  aggregated_cases?: number;
  cases_population_ratio?: number;
}

export interface AnalyticsAPIResponse extends APIResponse {
  data?: AnalyticsAPIData;
}

export interface SystemDateResponse extends APIResponse {
  system_date?: string;
}

// Mirror of utils/pydantic_model/request.py (query params)

export type Interval = "1" | "3" | "7" | "14" | "30";

export interface NationalParams {
  now: string;
  interval: Interval;
}

export interface CityParams extends NationalParams {
  city: string;
  ratio?: boolean;
}

export interface RegionParams extends CityParams {
  region: string;
}
