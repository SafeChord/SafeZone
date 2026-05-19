import { useRef, useEffect, useState, useCallback } from "react";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import type { CityRow } from "@/hooks/useCases";
import { fetchRegionCases } from "@/services/caseService";
import type { Interval } from "@/types/api";

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

const TAIWAN_CENTER: [number, number] = [120.5855, 23.5525];
const TAIWAN_ZOOM = 6.3;
const FIT_PADDING = 20;

function geojsonBounds(geojson: GeoJSON.FeatureCollection): maplibregl.LngLatBoundsLike | null {
  let minLng = Infinity, minLat = Infinity, maxLng = -Infinity, maxLat = -Infinity;
  for (const feat of geojson.features) {
    const geom = feat.geometry;
    const rings = geom.type === "Polygon"
      ? (geom as GeoJSON.Polygon).coordinates
      : geom.type === "MultiPolygon"
        ? (geom as GeoJSON.MultiPolygon).coordinates.flat()
        : [];
    for (const ring of rings) {
      for (const coord of ring) {
        const lng = coord[0] as number;
        const lat = coord[1] as number;
        if (lng < minLng) minLng = lng;
        if (lng > maxLng) maxLng = lng;
        if (lat < minLat) minLat = lat;
        if (lat > maxLat) maxLat = lat;
      }
    }
  }
  if (!isFinite(minLng)) return null;
  return [[minLng, minLat], [maxLng, maxLat]];
}

// Plotly "Reds" colorscale approximation (7 stops)
const REDS: [number, number, number][] = [
  [255, 245, 240], // #fff5f0
  [254, 224, 210], // #fee0d2
  [252, 174, 145], // #fcae91
  [251, 106, 74],  // #fb6a4a
  [239, 59, 44],   // #ef3b2c
  [203, 24, 29],   // #cb181d
  [103, 0, 13],    // #67000d
];

function getChoroplethColor(value: number, max: number): string {
  if (max === 0 || value <= 0) return "rgb(255,245,240)";
  // Log scale: compress high end, expand low end so small non-zero values are visible
  const t = Math.log1p(value) / Math.log1p(max);
  const pos = t * (REDS.length - 1);
  const lo = Math.floor(pos);
  const hi = Math.min(lo + 1, REDS.length - 1);
  const f = pos - lo;
  const cLo = REDS[lo]!;
  const cHi = REDS[hi]!;
  const r = Math.round(cLo[0] + (cHi[0] - cLo[0]) * f);
  const g = Math.round(cLo[1] + (cHi[1] - cLo[1]) * f);
  const b = Math.round(cLo[2] + (cHi[2] - cLo[2]) * f);
  return `rgb(${r},${g},${b})`;
}

export function RiskMap({ cities, ratio, systemDate, interval }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const [cityCode, setCityCode] = useState<Record<string, string>>({});
  // layer 0 = Taiwan (city-level), layer 1 = drilled into a city (region-level)
  const [drillCity, setDrillCity] = useState<string | null>(null);
  const [regionData, setRegionData] = useState<RegionRow[]>([]);

  // Load reference data
  useEffect(() => {
    fetch(import.meta.env.BASE_URL + "geo/city_code.json")
      .then((r) => r.json())
      .then((codes) => setCityCode(codes as Record<string, string>));
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
      const cityGeoUrl = import.meta.env.BASE_URL + "geo/geo_city.json";
      map.addSource("cities", {
        type: "geojson",
        data: cityGeoUrl,
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
      if (!map || !code || !systemDate) return;

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

      // Apply choropleth colors immediately (before useEffect, to avoid isStyleLoaded race)
      if (rows.length > 0) {
        const maxVal = Math.max(
          ...rows.map((r) => (ratio ? (r.ratio ?? 0) : r.cases)),
          1,
        );
        const parts: (string | string[])[] = [];
        for (const row of rows) {
          parts.push(row.region, getChoroplethColor(ratio ? (row.ratio ?? 0) : row.cases, maxVal));
        }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const matchExpr: any = ["match", ["get", "TOWNNAME"], ...parts, "#f0f0f0"];
        map.setPaintProperty("region-fill", "fill-color", matchExpr);
      }

      // Show region layers, hide city layers
      map.setLayoutProperty("city-fill", "visibility", "none");
      map.setLayoutProperty("city-outline", "visibility", "none");
      map.setLayoutProperty("region-fill", "visibility", "visible");
      map.setLayoutProperty("region-outline", "visibility", "visible");

      // Fit to region bounds
      const regionBounds = geojsonBounds(geoJson);
      if (regionBounds) {
        map.fitBounds(regionBounds, { padding: FIT_PADDING });
      }

      setDrillCity(cityName);
      setRegionData(rows);
    },
    [cityCode, systemDate, interval, ratio],
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

  // Hover tooltip
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    const popup = new maplibregl.Popup({ closeButton: false, closeOnClick: false, offset: 10 });

    const handleCityHover = (e: maplibregl.MapMouseEvent) => {
      const features = map.queryRenderedFeatures(e.point, { layers: ["city-fill"] });
      if (!features.length) { popup.remove(); return; }
      const name = features[0]!.properties?.["COUNTYNAME"] as string | undefined;
      if (!name) return;
      const row = cities.find((c) => c.city === name);
      const val = row ? (ratio ? (row.ratio ?? 0) : row.cases) : 0;
      const label = ratio ? `${val.toFixed(2)} / 萬人` : val.toLocaleString();
      popup.setLngLat(e.lngLat).setHTML(`<strong>${name}</strong><br/>${label}`).addTo(map);
    };

    const handleRegionHover = (e: maplibregl.MapMouseEvent) => {
      const features = map.queryRenderedFeatures(e.point, { layers: ["region-fill"] });
      if (!features.length) { popup.remove(); return; }
      const name = features[0]!.properties?.["TOWNNAME"] as string | undefined;
      if (!name) return;
      const row = regionData.find((r) => r.region === name);
      const val = row ? (ratio ? (row.ratio ?? 0) : row.cases) : 0;
      const label = ratio ? `${val.toFixed(2)} / 萬人` : val.toLocaleString();
      popup.setLngLat(e.lngLat).setHTML(`<strong>${name}</strong><br/>${label}`).addTo(map);
    };

    const clearPopup = () => popup.remove();

    map.on("mousemove", "city-fill", handleCityHover);
    map.on("mouseleave", "city-fill", clearPopup);
    map.on("mousemove", "region-fill", handleRegionHover);
    map.on("mouseleave", "region-fill", clearPopup);

    return () => {
      popup.remove();
      map.off("mousemove", "city-fill", handleCityHover);
      map.off("mouseleave", "city-fill", clearPopup);
      map.off("mousemove", "region-fill", handleRegionHover);
      map.off("mouseleave", "region-fill", clearPopup);
    };
  }, [cities, regionData, ratio]);

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
