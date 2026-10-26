import logging

from dash import Output, Input # type: ignore

from components.card import get_cases_card, get_top_cities_card


logger = logging.getLogger(__name__)

# updating the cardbody of following cards
# dbc.Card(None, className="text-white bg-primary mb-3", id="cases-card"),
# dbc.Card(None, className="mb-3", id="top-cities-card"),

def card_callbacks(app):
    @app.callback(
        Output("cases-card", "children"),
        Input("system-date-store", "data"),
        prevent_initial_call=True,
    )
    def update_national_cases(date_data):
        return get_cases_card(date_data["system_date"])

    @app.callback(
        Output("top-cities-card", "children"),
        Input("system-date-store", "data"),
    )
    def update_city_cases(date_data):
        return get_top_cities_card(date_data["system_date"])