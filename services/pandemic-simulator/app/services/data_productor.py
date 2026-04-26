# app/services/data_productor.py
import pandas as pd # type: ignore

from exceptions.custom import EmptyDataError

def read_csv():
    data = pd.read_csv("/data/covid_data.csv")

    # Keep only useful information in the data
    selected_columns = ["個案研判日", "縣市", "鄉鎮", "確定病例數"]
    data = data[selected_columns]
    # rename data columns
    data.rename(
        columns={
            "個案研判日": "date",
            "縣市": "city",
            "鄉鎮": "region",
            "確定病例數": "cases",
        },
        inplace=True,
    )
    # aggregate the data by date, city, and region
    data = data.groupby(["date", "city", "region"], as_index=False)["cases"].sum()
    
    # format the date in the data
    data["date"] = pd.to_datetime(data["date"], format="%Y/%m/%d")
    # Reformat the date to "%Y-%m-%d"
    data["date"] = data["date"].dt.strftime("%Y-%m-%d")

    return data


def get_data_by_date(date):
    data = read_csv()
    # Filter data for a specific date
    filtered_data = data[data["date"] == date]
    if filtered_data.empty:
        raise EmptyDataError
    return filtered_data.to_dict(orient="records")


def get_data_by_interval(start_date, end_date):
    data = read_csv()
    # Filter data for a specific date range
    filtered_data = data[(data["date"] >= start_date) & (data["date"] <= end_date)]
    if filtered_data.empty:
        raise EmptyDataError
    return filtered_data.to_dict(orient="records")
