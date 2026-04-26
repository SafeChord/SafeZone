# app/services/data_sender.py
import logging
import asyncio

import httpx  # type: ignore
from pydantic import ValidationError  # type: ignore

from utils.pydantic_model.request import CovidDataModel
from utils.context import trace_id_var

from exceptions.custom import ServiceValidationError
from config.settings import INGESTOR_URL, MAX_CONCURRENT_REQUESTS


logger = logging.getLogger(__name__)
logging.getLogger("httpx").setLevel(logging.WARNING)

async def send_one(sem, client, url, case):
    async with sem:
        trace_header = {"X-Trace-ID": str(trace_id_var.get())}
        await client.post(url, headers=trace_header, json=case, timeout=5.0)
        logger.info(f"Case = {case} sent to ingestor.", extra={"event": "case_sent"})


async def send_data(data):
    # add semaphore to limit the number of concurrent requests
    sem = asyncio.Semaphore(MAX_CONCURRENT_REQUESTS)
    # validate all data first, raise at once if invalid
    try:
        payloads = [CovidDataModel(**case).model_dump() for case in data]
    except ValidationError as e:
        raise ServiceValidationError(errors=e)

    async with httpx.AsyncClient() as client:
        tasks = [
            send_one(sem, client, INGESTOR_URL + "/covid_event", payload)
            for payload in payloads
        ]
        # gather 所有併發請求
        await asyncio.gather(*tasks)
