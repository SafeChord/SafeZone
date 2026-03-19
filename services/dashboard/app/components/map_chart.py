import json

import plotly.graph_objects as go  # type: ignore
import dash_bootstrap_components as dbc  # type: ignore
from dash import dcc, html  # type: ignore


def get_city_geodata(city):
    # load the geojson data for the specified city

    # load the city center location
    with open("/app/utils/geo_data/boundaries/accessory/city_center.json", "r") as f:
        city_center = json.load(f)

    # load the city code mapping
    with open("/app/utils/geo_data/boundaries/accessory/city_code.json", "r") as f:
        city_code_mapping = json.load(f)
    # load the geojson data by city code
    city_code = city_code_mapping[city]
    with open(
        f"/app/utils/geo_data/boundaries/regions/{city_code}_region.json",
        "r",
    ) as f:
        geojson_data = json.load(f)
    return {
        "center": city_center[city],
        "geojson": geojson_data,
    }


# risk map canvas
def get_init_map():
    # map banner
    banner = html.H4(
        f" 1970-01-01 ~ 1970-01-03 疫情風險圖 - 依病例數",
        id="risk-map-title",
        className="text-white bg-primary p-2",
    )
    # the current risk map
    risk_map = dcc.Graph(
        id="risk-map",
        figure=go.Figure(),
    )
    # store the map current layer and location
    risk_map_state = dcc.Store(
        id="risk-map-state",
        data={
            "layer": "0",  # 0: tw risk map, 1: city risk map
            "date": "1970-01-03", 
            "interval": "3", # interval in days
            "show": "cases", # cases or ratio
            "loc": "台灣",  # current location, default is Taiwan
        },
    )
    # the map cache
    map_cache = dcc.Store(
        id="map-cache",
        data=None,
    )
    # store the filters state of map in cache
    map_cache_state = dcc.Store(
        id="map-cache-state",
        data={
            "layer": "0",
            "date": "1970-01-03",
            "interval": "3",
            "show": "cases",
            "loc": "台灣",
        },
    )

    return dbc.Row(
        [
            banner,
            risk_map_state,
            risk_map,
            map_cache_state,
            map_cache,
        ]
    )


# risk map of whole taiwan
def get_tw_risk_map(city_risk):
    # risk map of whole taiwan with city level risk initialized
    with open(f"/app/utils/geo_data/boundaries/geo_city.json", "r") as f:
        geojson_data = json.load(f)

    return go.Figure(
        go.Choroplethmapbox(
            geojson=geojson_data,
            locations=list(city_risk.keys()),
            featureidkey="properties.COUNTYNAME",
            z=list(city_risk.values()),
            colorscale="Reds",
            marker_opacity=0.9,
            marker_line_width=1,
            showscale=False,
        ),
        layout={
            "mapbox": {
                "style": "carto-positron",
                "zoom": 6.3,
                "center": {"lat": 23.5525, "lon": 120.5855},  
            },
            "margin": {"r": 0, "t": 0, "l": 0, "b": 0},
            "uirevision": "tw",
        },
    )


# risk map of specific city
def get_city_risk_map(city, region_risk):
    # get the geojson data and center location for the specified city
    geo_data = get_city_geodata(city)

    figure = go.Figure(
        go.Choroplethmapbox(
            geojson=geo_data["geojson"],
            locations=list(region_risk.keys()),
            featureidkey="properties.TOWNNAME",
            z=list(region_risk.values()),
            colorscale="Reds",
            marker_opacity=0.9,
            marker_line_width=1,
            showscale=False,
        ),
        layout={
            "mapbox": {
                "style": "carto-positron",
                "zoom": 8,
                "center": geo_data["center"],
            },
            "margin": {"r": 0, "t": 0, "l": 0, "b": 0},
            "uirevision": city,
        },
    )
    # add annotation to the figure
    figure.update_layout(
        annotations=[
            dict(
                text="點擊地圖任區域可返回全台地圖",
                x=0.03,
                y=0.95,
                xref="paper",
                yref="paper",
                showarrow=False,
                font=dict(size=20, color="red"),
                align="center",
                bgcolor="rgba(255, 255, 255, 0.7)",
                bordercolor="gray",
                borderwidth=1,
            )
        ]
    )
    return figure
