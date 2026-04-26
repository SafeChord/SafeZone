import asyncio
from pipeline.data_productor import get_data_by_date, get_data_by_interval
from pipeline.data_sender import send_data


async def handle_request(start_date, end_date=None):
    if end_date:
        filtered_data = get_data_by_interval(start_date, end_date)
    else:
        filtered_data = get_data_by_date(start_date)

    await send_data(filtered_data)
