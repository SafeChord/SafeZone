from datetime import datetime, timedelta

from dash import html  # type: ignore
import dash_bootstrap_components as dbc  # type: ignore

from services.update_cases import get_national_cases
from components.trend_chart import get_trend_bar


def get_cases_card(system_date: str):
    return dbc.CardBody(
        [
            html.H3(
                html.B("近七日新增案例"),
                className="card-title",
            ),
            html.H2(
                get_national_cases(date=system_date, interval="7"),
                style={"textAlign": "right"},
                className="card-text text-white bg-success",
            ),
            html.H3(
                html.B("今日新增案例"),
                className="card-title",
            ),
            html.H2(
                get_national_cases(date=system_date, interval="1"),
                style={"textAlign": "right"},
                className="card-text text-white bg-success",
            ),
        ]
    )


def get_top_cities_card(system_date: str):
    start_date = datetime.strptime(system_date, "%Y-%m-%d") - timedelta(days=7)
    start_date = start_date.strftime("%Y-%m-%d")

    return dbc.CardBody(
        [
            html.H4("最快案例成長城市前十名", className="card-title text-primary"),
            html.P(
                f"統計日期為{start_date} ~ {system_date}",
                className="card-subtitle text-primary",
            ),
            get_trend_bar(date=system_date),
        ]
    )
