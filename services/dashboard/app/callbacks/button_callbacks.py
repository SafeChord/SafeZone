import logging

from dash import Input, Output # type: ignore


logger = logging.getLogger(__name__)


def interval_button_callbacks(app):
    @app.callback(
        [
            Output("btn-3-days", "active"),
            Output("btn-7-days", "active"),
            Output("btn-14-days", "active"),
            Output("btn-30-days", "active"),
            Output("interval-button-state", "data"),
        ],
        [
            Input("btn-3-days", "n_clicks_timestamp"),
            Input("btn-7-days", "n_clicks_timestamp"),
            Input("btn-14-days", "n_clicks_timestamp"),
            Input("btn-30-days", "n_clicks_timestamp"),
        ]
        ,
        prevent_initial_call=True,
    )
    def update_active_button(ts1, ts2, ts3, ts4):
        timestamps = {
            "btn-3-days": ts1,
            "btn-7-days": ts2,
            "btn-14-days": ts3,
            "btn-30-days": ts4,
        }

        # find the newest clicked button
        active_button = max(timestamps, key=lambda x: timestamps[x] or 0)

        logger.debug(f"Interval active button is {active_button}.")

        # return the active state of each button
        return [
            active_button == "btn-3-days",
            active_button == "btn-7-days",
            active_button == "btn-14-days",
            active_button == "btn-30-days",
            {"active": active_button.split("-")[1]},
        ]


def ratio_button_callbacks(app):
    @app.callback(
        [
            Output("btn-cases", "active"),
            Output("btn-ratio", "active"),
            Output("ratio-button-state", "data"),
        ],
        [
            Input("btn-cases", "n_clicks_timestamp"),
            Input("btn-ratio", "n_clicks_timestamp"),
        ],
        prevent_initial_call=True,
    )
    def update_active_button(ts1, ts2):
        timestamps = {
            "btn-cases": ts1,
            "btn-ratio": ts2,
        }

        # find the newest clicked button
        active_button = max(timestamps, key=lambda x: timestamps[x] or 0)

        logger.debug(f"Ratio active button is {active_button}.")

        # return the active state of each button
        return [
            active_button == "btn-cases",
            active_button == "btn-ratio",
            {"active": active_button.split("-")[1]},
        ]
