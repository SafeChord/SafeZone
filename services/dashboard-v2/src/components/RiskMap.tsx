import { useRef, useEffect, useState, useCallback } from "react";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import type { CityRow } from "@/hooks/useCases";
import { fetchRegionCases } from "@/services/caseService";
import type { Interval } from "@/types/api";

interface CityCenter {
  lat: number;
  lon: number;
}

interface RegionRow {
  region: string;
  cases: number;
  ratio?: number;
}

interface Props {
  cities: CityRow[];
  ratio: boolean;
  systemDate: string | null;
  interval: Interval;
}

const TAIWAN_CENTER: [number, number] = [120.9, 23.7];
const TAIWAN_ZOOM = 7;
const CITY_ZOOM = 10;

function getChoroplethColor(value: number, max: number): string {
  if (max === 0) return "#f0f0f0";
  const t = Math.min(value / max, 1);
  const r = Math.round(240 + (220 - 240) * t);
  const g = Math.round(240 + (50 - 240) * t);
  const b = Math.round(240 + (47 - 240) * t);
  return `rgb(${r},${g},${b})`;
}

export function RiskMap({ cities, ratio, systemDate, interval }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const [cityCode, setCityCode] = useState<Record<string, string>>({});
  const [cityCenter, setCityCenter] = useState<Record<string, CityCenter>>({});
  // layer 0 = Taiwan (city-level), layer 1 = drilled into a city (region-level)
  const [drillCity, setDrillCity] = useState<string | null>(null);
  const [regionData, setRegionData] = useState<RegionRow[]>([]);

  // Load reference data
  useEffect(() => {
    Promise.all([
      fetch(import.meta.env.BASE_URL + "geo/city_code.json").then((r) => r.json()),
      fetch(import.meta.env.BASE_URL + "geo/city_center.json").then((r) => r.json()),
    ]).then(([codes, centers]) => {
      setCityCode(codes as Record<string, string>);
      setCityCenter(centers as Record<string, CityCenter>);
    });
  }, []);

  // Initialize map
  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    const map = new maplibregl.Map({
      container: containerRef.current,
      style: {
        version: 8,
        sources: {},
        layers: [
          {
            id: "background",
            type: "background",
            paint: { "background-color": "#e8f4f8" },
          },
        ],
      },
      center: TAIWAN_CENTER,
      zoom: TAIWAN_ZOOM,
      attributionControl: false,
    });

    map.addControl(new maplibregl.NavigationControl(), "top-right");

    map.on("load", () => {
      map.addSource("cities", {
        type: "geojson",
        data: import.meta.env.BASE_URL + "geo/geo_city.json",
      });

      map.addLayer({
        id: "city-fill",
        type: "fill",
        source: "cities",
        paint: { "fill-color": "#f0f0f0", "fill-opacity": 0.8 },
      });

      map.addLayer({
        id: "city-outline",
        type: "line",
        source: "cities",
        paint: { "line-color": "#999", "line-width": 1 },
      });

      // Region layer (initially empty, populated on drill-down)
      map.addSource("regions", {
        type: "geojson",
        data: { type: "FeatureCollection", features: [] },
      });

      map.addLayer({
        id: "region-fill",
        type: "fill",
        source: "regions",
        paint: { "fill-color": "#f0f0f0", "fill-opacity": 0.8 },
      });

      map.addLayer({
        id: "region-outline",
        type: "line",
        source: "regions",
        paint: { "line-color": "#999", "line-width": 1 },
      });

      // Initially hide region layers
      map.setLayoutProperty("region-fill", "visibility", "none");
      map.setLayoutProperty("region-outline", "visibility", "none");
    });

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  // Drill down into city
  const drillInto = useCallback(
    async (cityName: string) => {
      const map = mapRef.current;
      const code = cityCode[cityName];
      const center = cityCenter[cityName];
      if (!map || !code || !center || !systemDate) return;

      // Load region GeoJSON
      const geoRes = await fetch(
        import.meta.env.BASE_URL + `geo/regions/${code}_region.json`,
      );
      const geoJson = await geoRes.json();

      // Load region case data
      const regionNames: string[] = geoJson.features.map(
        (f: { properties: { TOWNNAME: string } }) => f.properties.TOWNNAME,
      );

      const results = await Promise.all(
        regionNames.map((region) =>
          fetchRegionCases(systemDate, interval, cityName, region, ratio),
        ),
      );

      const rows: RegionRow[] = results
        .filter((r) => r.success && r.data)
        .map((r) => ({
          region: r.data!.region!,
          cases: r.data!.aggregated_cases ?? 0,
          ratio: r.data!.cases_population_ratio,
        }));

      // Update region source
      (map.getSource("regions") as maplibregl.GeoJSONSource).setData(geoJson);

      // Show region layers, hide city layers
      map.setLayoutProperty("city-fill", "visibility", "none");
      map.setLayoutProperty("city-outline", "visibility", "none");
      map.setLayoutProperty("region-fill", "visibility", "visible");
      map.setLayoutProperty("region-outline", "visibility", "visible");

      // Fly to city
      map.flyTo({ center: [center.lon, center.lat], zoom: CITY_ZOOM });

      setDrillCity(cityName);
      setRegionData(rows);
    },
    [cityCode, cityCenter, systemDate, interval, ratio],
  );

  // Drill back to Taiwan view
  const drillBack = useCallback(() => {
    const map = mapRef.current;
    if (!map) return;

    map.setLayoutProperty("city-fill", "visibility", "visible");
    map.setLayoutProperty("city-outline", "visibility", "visible");
    map.setLayoutProperty("region-fill", "visibility", "none");
    map.setLayoutProperty("region-outline", "visibility", "none");

    map.flyTo({ center: TAIWAN_CENTER, zoom: TAIWAN_ZOOM });

    setDrillCity(null);
    setRegionData([]);
  }, []);

  // Reset to Taiwan view when filters change
  useEffect(() => {
    if (drillCity) {
      drillBack();
    }
  }, [interval, ratio]);

  // Update city choropleth colors
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !map.isStyleLoaded() || cities.length === 0) return;
    if (Object.keys(cityCode).length === 0) return;

    const valueByCode = new Map<string, number>();
    for (const row of cities) {
      const code = cityCode[row.city];
      if (!code) continue;
      valueByCode.set(code, ratio ? (row.ratio ?? 0) : row.cases);
    }

    const maxVal = Math.max(...valueByCode.values(), 1);

    const parts: (string | maplibregl.ExpressionSpecification)[] = [];
    for (const [code, val] of valueByCode) {
      parts.push(code, getChoroplethColor(val, maxVal));
    }

    if (parts.length === 0) return;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const matchExpr: any = ["match", ["get", "COUNTYCODE"], ...parts, "#f0f0f0"];
    map.setPaintProperty("city-fill", "fill-color", matchExpr);
  }, [cities, ratio, cityCode]);

  // Update region choropleth colors
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !map.isStyleLoaded() || regionData.length === 0) return;

    const maxVal = Math.max(
      ...regionData.map((r) => (ratio ? (r.ratio ?? 0) : r.cases)),
      1,
    );

    const parts: (string | maplibregl.ExpressionSpecification)[] = [];
    for (const row of regionData) {
      parts.push(row.region, getChoroplethColor(ratio ? (row.ratio ?? 0) : row.cases, maxVal));
    }

    if (parts.length === 0) return;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const matchExpr: any = ["match", ["get", "TOWNNAME"], ...parts, "#f0f0f0"];
    map.setPaintProperty("region-fill", "fill-color", matchExpr);
  }, [regionData, ratio]);

  // Click handlers
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    const handleCityClick = (e: maplibregl.MapMouseEvent) => {
      const features = map.queryRenderedFeatures(e.point, { layers: ["city-fill"] });
      if (!features.length) return;
      const name = features[0]!.properties?.["COUNTYNAME"] as string | undefined;
      if (name) drillInto(name);
    };

    const handleRegionClick = () => {
      drillBack();
    };

    map.on("click", "city-fill", handleCityClick);
    map.on("click", "region-fill", handleRegionClick);
    map.on("mouseenter", "city-fill", () => { map.getCanvas().style.cursor = "pointer"; });
    map.on("mouseleave", "city-fill", () => { map.getCanvas().style.cursor = ""; });
    map.on("mouseenter", "region-fill", () => { map.getCanvas().style.cursor = "pointer"; });
    map.on("mouseleave", "region-fill", () => { map.getCanvas().style.cursor = ""; });

    return () => {
      map.off("click", "city-fill", handleCityClick);
      map.off("click", "region-fill", handleRegionClick);
    };
  }, [drillInto, drillBack]);

  return (
    <div style={{ position: "relative" }}>
      {drillCity && (
        <div
          style={{
            position: "absolute",
            top: "10px",
            left: "10px",
            zIndex: 1,
            background: "rgba(255,255,255,0.9)",
            border: "1px solid #e57373",
            borderRadius: "4px",
            padding: "6px 12px",
            fontSize: "13px",
            color: "#c62828",
          }}
        >
          {drillCity} — click any region to return
        </div>
      )}
      <div
        ref={containerRef}
        style={{ width: "100%", height: "100%", minHeight: "500px", borderRadius: "8px" }}
      />
    </div>
  );
}
