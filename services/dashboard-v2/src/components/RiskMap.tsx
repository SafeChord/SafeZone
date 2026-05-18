import { useRef, useEffect, useState } from "react";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import type { CityRow } from "@/hooks/useCases";

interface CityCenter {
  lat: number;
  lon: number;
}

interface Props {
  cities: CityRow[];
  ratio: boolean;
}

const TAIWAN_CENTER: [number, number] = [120.9, 23.7];
const INITIAL_ZOOM = 7;

function getChoroplethColor(value: number, max: number): string {
  if (max === 0) return "#f0f0f0";
  const t = Math.min(value / max, 1);
  const r = Math.round(240 + (220 - 240) * t);
  const g = Math.round(240 + (50 - 240) * t);
  const b = Math.round(240 + (47 - 240) * t);
  return `rgb(${r},${g},${b})`;
}

export function RiskMap({ cities, ratio }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const [cityCode, setCityCode] = useState<Record<string, string>>({});
  const [cityCenter, setCityCenter] = useState<Record<string, CityCenter>>({});

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
      zoom: INITIAL_ZOOM,
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
        paint: {
          "fill-color": "#f0f0f0",
          "fill-opacity": 0.8,
        },
      });

      map.addLayer({
        id: "city-outline",
        type: "line",
        source: "cities",
        paint: {
          "line-color": "#999",
          "line-width": 1,
        },
      });
    });

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  // Update choropleth colors when data changes
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

    // Build match expression dynamically
    const parts: (string | maplibregl.ExpressionSpecification)[] = [];
    for (const [code, val] of valueByCode) {
      parts.push(code, getChoroplethColor(val, maxVal));
    }

    if (parts.length === 0) return;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const matchExpr: any = ["match", ["get", "COUNTYCODE"], ...parts, "#f0f0f0"];
    map.setPaintProperty("city-fill", "fill-color", matchExpr);
  }, [cities, ratio, cityCode]);

  // Popup on click
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    const handleClick = (e: maplibregl.MapMouseEvent) => {
      const features = map.queryRenderedFeatures(e.point, { layers: ["city-fill"] });
      if (!features.length) return;

      const props = features[0]!.properties;
      const name = props?.["COUNTYNAME"] as string | undefined;
      if (!name) return;

      const row = cities.find((c) => c.city === name);
      const center = cityCenter[name];

      const html = row
        ? `<strong>${name}</strong><br/>Cases: ${row.cases.toLocaleString()}${row.ratio != null ? `<br/>Ratio: ${row.ratio.toFixed(4)}%` : ""}`
        : `<strong>${name}</strong><br/>No data`;

      const lngLat: [number, number] = center
        ? [center.lon, center.lat]
        : [e.lngLat.lng, e.lngLat.lat];

      new maplibregl.Popup({ closeButton: false })
        .setLngLat(lngLat)
        .setHTML(html)
        .addTo(map);
    };

    map.on("click", "city-fill", handleClick);
    map.on("mouseenter", "city-fill", () => { map.getCanvas().style.cursor = "pointer"; });
    map.on("mouseleave", "city-fill", () => { map.getCanvas().style.cursor = ""; });

    return () => {
      map.off("click", "city-fill", handleClick);
    };
  }, [cities, cityCenter]);

  return (
    <div
      ref={containerRef}
      style={{ width: "100%", height: "100%", minHeight: "500px", borderRadius: "8px" }}
    />
  );
}
