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

counter = meter.create_counter("counter", description="Counting of processed valuables")


def get_env():
    _ = load_dotenv(PROJECT_ROOT / ".env")


def get_url() -> str:
    host = os.getenv("RABBITMQ_HOST")
    user = os.getenv("RABBITMQ_DEFAULT_USER")
    pwd = os.getenv("RABBITMQ_DEFAULT_PASS")
    port = os.getenv("RABBITMQ_PORT")

    url = f"amqp://{user}:{pwd}@{host}:{port}/"
    return url


@tracer.start_as_current_span("get_data_info")
async def get_data(
    url: str, client: httpx.AsyncClient, cfg: HttpConfig
) -> httpx.Response | None:

    try:
        span = trace.get_current_span()
        span.set_attribute("fetch.url", url)

        logger.info(f"Start fetching data from {url}")
        response = await client.get(url)
        if response.status_code == 404:
            logger.warning(f"{url} is not exist")
            return None

        if response.status_code == 429:
            span.add_event("rate limited", {"url": url, "status_code": 429})
            logger.warning(f"Rate limited (429) while fetching {url}, starting exponential backoff")

            response = await retry_on_rate_limit(url, client, cfg)
            if response is None:
                logger.error(f"Failed to fetch data after retries for {url}")
                return None

        counter.add(1, {"url": url, "stage": "utils", "type": "timeout.errors"})

    # TODO: створити власний ексепшн
    except httpx.TimeoutException:
        logger.error(f"Timeout while fetching data from {url}")
        return None

    logger.info(f"Fetched data from {url}")
    return response


@tracer.start_as_current_span("retry_get_data_info")
async def again_with_retries(
    url: str, client: httpx.AsyncClient, cfg: HttpConfig
) -> httpx.Response | None:

    retries = cfg.max_retries
    interval = get_random_interval(cfg.interval, cfg.min_delay, cfg.max_delay)
    while retries > 0:
        response = await client.get(url, timeout=cfg.timeout)
        if response.status_code == 200:
            return response

        retries -= 1
        await asyncio.sleep(interval)

    return None


async def retry_on_rate_limit(
    url: str, client: httpx.AsyncClient, cfg: HttpConfig
) -> httpx.Response | None:

    for attempt in range(cfg.max_retries):
        delay = 2 ** attempt
        await asyncio.sleep(delay)

        response = await client.get(url, timeout=cfg.timeout)
        if response.status_code == 200:
            return response

    return None


def get_random_interval(interval: float, min: float, max: float) -> float:
    return interval + random.uniform(min, max)
