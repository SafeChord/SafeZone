# /bin/client.py
import time
import json
import uuid
import logging
from typing import Optional

import requests  # type: ignore
from pathlib import Path  # type: ignore
from pydantic import BaseModel  # type: ignore

from utils.pydantic_model.request import (
    SimulateModel,
    VerifyModel,
    SetTimeModel,
    HealthCheckModel,
    DBInitModel,
)
from utils.pydantic_model.response import (
    APIResponse,
    HealthResponse,
    SystemDateResponse,
    MocktimeStatusResponse,
    AnalyticsAPIResponse,
)
from config.settings import RELAY_URL, RELAY_TIMEOUT
from config.settings import TOKEN_FILE, CLIENT_ID, CLIENT_SECRET, REFRESH_TOKEN
from bin.context import global_context

logger = logging.getLogger(__name__)


class BaseAuthClient:
    def __init__(self):
        self._refresh_token()

    def _refresh_token(self) -> str:
        logger.debug("Refreshing authentication token...")
        # re-authenticate if token is missing or expired
        payload = {
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "refresh_token": REFRESH_TOKEN,
            "grant_type": "refresh_token",
        }
        logger.debug(f"Payload for token refresh: {payload}")
        resp = requests.post("https://oauth2.googleapis.com/token", data=payload)
        token_data = resp.json()

        # delete old token file if exists
        if Path(TOKEN_FILE).exists():
            Path(TOKEN_FILE).unlink()

        # save the new token to file
        with open(TOKEN_FILE, "w") as f:
            token_save = {
                "id_token": token_data["id_token"],
                "exp": time.time() + token_data["expires_in"],
            }
            json.dump(token_save, f)
        logger.debug(f"New token saved to {TOKEN_FILE}")

        return token_data["id_token"]

    def _get_id_token(self) -> str:
        # if token is missing
        if not Path(TOKEN_FILE).exists():
            return self._refresh_token()
        with open(TOKEN_FILE) as f:
            token_data = json.load(f)
            if time.time() < token_data["exp"]:
                return token_data["id_token"]
        # if token is expired
        return self._refresh_token()

    def _get_header(self):
        header = {"content-type": "application/json"}
        # generate the authentication header
        header["Authorization"] = f"Bearer {self._get_id_token()}"
        # generate the trace header
        header["X-Trace-ID"] = global_context.get("trace_id", str(uuid.uuid4()))
        return header

    def auth_request(
        self,
        method: str,
        path: str,
        payload: Optional[dict] = None,
        params: Optional[dict] = None,
        request_model: BaseModel = None,
        response_model: BaseModel = APIResponse,
    ) -> dict:
        logger.debug(
            f"[Request] {method} {RELAY_URL}/{path} with payload: {payload} and params: {params}"
        )
        if method in ["POST", "PUT", "PATCH"] and request_model and payload:
            # Validate the payload against the request model if provided
            payload = request_model(**payload).model_dump(mode="json")

        elif method in ["GET", "DELETE"] and request_model and params:
            # Validate the params against the request model if provided
            params = request_model(**params).model_dump(mode="json")

        resp = requests.request(
            method=method,
            url=f"{RELAY_URL}/{path}",
            headers=self._get_header(),
            json=(
                payload if method not in ["GET", "DELETE"] else None
            ),  # Adjust based on methods that don't typically have a body
            params=params,
            timeout=RELAY_TIMEOUT,
        )
        resp.raise_for_status()
        resp_body = resp.json()
        checked_resp = response_model(**resp_body)

        result = checked_resp.model_dump()
        result["headers"] = dict(resp.headers)
        return result


class DataflowClient(BaseAuthClient):
    def simulate(self, **kwargs):
        return self.auth_request(
            method="POST",
            path="dataflow/simulate",
            payload=kwargs,
            request_model=SimulateModel,
        )

    def verify(self, **kwargs):
        return self.auth_request(
            method="GET",
            path="dataflow/verify",
            params=kwargs,
            request_model=VerifyModel,
            response_model=AnalyticsAPIResponse,
        )


class TimeClient(BaseAuthClient):
    def now(self):
        return self.auth_request(
            method="GET", path="system/time/now", response_model=SystemDateResponse
        )

    def set(self, **kwargs):
        return self.auth_request(
            method="POST",
            path="system/time/set",
            payload=SetTimeModel(**kwargs).model_dump(mode="json"),
            request_model=SetTimeModel,
        )

    def get_status(self):
        return self.auth_request(
            method="GET",
            path="system/time/status",
            response_model=MocktimeStatusResponse,
        )


class HealthClient(BaseAuthClient):
    def check(self, **kwargs):
        return self.auth_request(
            method="GET",
            path="system/health",
            params=kwargs,
            request_model=HealthCheckModel,
            response_model=HealthResponse,
        )


class DBClient(BaseAuthClient):
    def init(self, force: bool = False):
        return self.auth_request(
            method="POST",
            path="db/init",
            payload=DBInitModel(force=force).model_dump(mode="json"),
            request_model=DBInitModel,
        )

    def clear(self):
        return self.auth_request(method="POST", path="db/clear")

    def reset(self):
        return self.auth_request(method="POST", path="db/reset")
