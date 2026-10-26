import logging
from datetime import datetime, timedelta

import dash  # type: ignore
from dash import Output, Input, State  # type: ignore

from components.map_chart import get_tw_risk_map, get_city_risk_map
from services.update_cases import get_region_data, get_city_data

logger = logging.getLogger(__name__)


def risk_title_callbacks(app):
    @app.callback(
        Output("risk-map-title", "children"),
        [
            Input("interval-button-state", "data"),
            Input("ratio-button-state", "data"),
            Input("system-date-store", "data"),
        ],
        prevent_initial_call=True,
        # prevent initial call to avoid unnecessary updates
    )
    def update_risk_title(interval, ratio, date_data):
        interval = interval["active"]
        ratio = ratio["active"]
        # update the title based on the active button
        if ratio == "ratio":
            ratio_str = "依比例"
        else:
            ratio_str = "依病例數"
        # update the start and end date based on the active button
        end_date = datetime.strptime(date_data["system_date"], "%Y-%m-%d")
        start_date = end_date - timedelta(days=int(interval) + 1)
        end_date = end_date.strftime("%Y-%m-%d")
        start_date = start_date.strftime("%Y-%m-%d")

        return f"{start_date} ~ {end_date} 疫情風險圖 - {ratio_str}"


last_click_data = None

def risk_map_callbacks(app):
    @app.callback(
        [
            Output("risk-map", "figure"),
            Output("risk-map-state", "data"),
            Output("map-cache", "data"),
            Output("map-cache-state", "data"),
        ],
        [
            Input("risk-map", "clickData"),
            Input("interval-button-state", "data"),
            Input("ratio-button-state", "data"),
            Input("system-date-store", "data"),
        ],
        [
            State("risk-map", "figure"),
            State("risk-map-state", "data"),
            State("map-cache", "data"),
            State("map-cache-state", "data"),
        ],
        prevent_initial_call=True,
        # prevent initial call to avoid unnecessary updates
    )
    def update_risk_map(
        click_map,
        interval,
        ratio,
        date_data,
        risk_map,
        map_state,
        map_cache,
        cache_state,
    ):
        update_state = {
            "layer": map_state["layer"],
            "date": map_state["date"],
            "interval": map_state["interval"],
            "show": map_state["show"],
            "loc": map_state["loc"],
        }

        if click_map:
            global last_click_data
            # if the click data is same as the last click data, do not update the map
            if click_map != last_click_data:
                logger.debug(f"Click data is changed: {click_map}")
                if update_state["layer"] == "0":
                    # if the map is a whole tw risk map, change it to city risk map
                    update_state["layer"] = "1"
                    update_state["loc"] = click_map["points"][0]["location"]
                elif update_state["layer"] == "1":
                    # if the map is a city risk map, change it to whole tw risk map
                    update_state["layer"] = "0"
                    update_state["loc"] = "台灣"
            last_click_data = click_map
        update_state["date"] = date_data["system_date"]
        update_state["interval"] = interval["active"]
        update_state["show"] = ratio["active"]

        logger.debug(f"Update map state: {update_state} from original state: {map_state}")

        if update_state == map_state:
            # if the map state is the same as the cache state return the cached map
            logger.info("update map with no changes, return cached map")
            return [
                dash.no_update,
                dash.no_update,
                dash.no_update,
                dash.no_update,
            ]
        elif update_state == cache_state:
            # if the map state is the same as the cache state return the cached map
            logger.info("update map is the same as cache state, return cached map")
            return [
                map_cache,
                cache_state,
                risk_map,
                map_state,
            ]
        else:
            # if the map state is not the same as the cache state, update the map
            logger.info("update map with changes")
            next_map = update_map(update_state)
            return [
                next_map,
                update_state,
                risk_map,
                map_state,
            ]



def update_map(curr_state):
    interval = curr_state["interval"]
    show = curr_state["show"]
    system_date = curr_state["date"]
    layer = curr_state["layer"]

    # layer == 0，whole tw risk map
    if layer == "0":
        # get city risk data
        city_risk = get_city_data(system_date, interval, show == "ratio")
        # create map
        return get_tw_risk_map(city_risk)

    # layer == 1，city risk map
    city = curr_state["loc"]
    region_risk = get_region_data(system_date, city, interval, show == "ratio")
    return get_city_risk_map(
        city=city,
        region_risk=region_risk,
    )
