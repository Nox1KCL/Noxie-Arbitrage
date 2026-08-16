import asyncio
import os
import random
from pathlib import Path
from opentelemetry import trace, metrics

import httpx
from dotenv import load_dotenv
from loguru import logger

from parsers.config.config import HttpConfig

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent.parent
tracer = trace.get_tracer(__name__)
meter = metrics.get_meter(__name__)

counter = meter.create_counter(
    "counter",
    description="Counting of processed valuables"
)

def get_env():
    _ = load_dotenv(PROJECT_ROOT / ".env")

def get_url() -> str:
    host = os.getenv("RABBIT_HOST")
    pwd = os.getenv("RABBIT_PWD")
    user = os.getenv("RABBIT_USER")
    port = os.getenv("RABBIT_PORT")

    url = f"amqp://{user}:{pwd}@{host}:{port}/"
    return url

@tracer.start_as_current_span("get_data_info")
async def get_data(url: str, client: httpx.AsyncClient, cfg: HttpConfig) -> httpx.Response:
    try:
        span = trace.get_current_span()
        span.set_attribute("fetch.url", url)

        logger.info(f"Start fetching data from {url}")
        response = await client.get(url)

        if response.status_code != 200:
            span.add_event("failed fetching url", {"url": url})
            logger.warning(f"Failed to fetch data from {url}, starting reattemts")

            retries = cfg.max_retries
            interval = get_random_interval(cfg.interval, cfg.min_delay, cfg.max_delay)
            while retries > 0:
                response = await client.get(url, timeout=cfg.timeout)
                if response.status_code == 200:
                    return response

                retries -= 1
                await asyncio.sleep(interval)
                
            counter.add(1, {"url": url, "stage": "utils", "type": "timeout.errors"})
            # TODO: створити власний ексепшн
            raise Exception(f"Failed to fetch data: {response.status_code}")

    except httpx.TimeoutException:
        raise Exception("Timeout while fetching data")

    logger.info(f"Fetched data from {url}")
    return response

def get_random_interval(interval: float, min: float, max: float) -> float:
    return interval + random.uniform(min, max)
