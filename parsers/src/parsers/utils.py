from dotenv import load_dotenv
import os, httpx, random, asyncio
from pathlib import Path
from loguru import logger

from parsers.config.config import HttpConfig

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent.parent

def get_env():
    _ = load_dotenv(PROJECT_ROOT / ".env")

def get_url() -> str:
    host = os.getenv("RABBIT_HOST")
    pwd = os.getenv("RABBIT_PWD")
    user = os.getenv("RABBIT_USER")
    port = os.getenv("RABBIT_PORT")

    url = f"amqp://{user}:{pwd}@{host}:{port}/"
    return url

async def get_data(url: str, client: httpx.AsyncClient, cfg: HttpConfig) -> httpx.Response:
    try:
        logger.info(f"Start fetching data from {url}")
        response = await client.get(url)

        if response.status_code != 200:
            logger.warning(f"Failed to fetch data from {url}, starting reattemts")
            retries = cfg.max_retries
            interval = get_random_interval(cfg.interval, cfg.min_delay, cfg.max_delay)
            while retries > 0:
                response = await client.get(url, timeout=cfg.timeout)
                if response.status_code == 200:
                    return response
                retries -= 1
                await asyncio.sleep(interval)
                
            # TODO: створити власний ексепшн
            raise Exception(f"Failed to fetch data: {response.status_code}")

    except httpx.TimeoutException:
        raise Exception("Timeout while fetching data")

    logger.info(f"Fetched data from {url}")
    return response

def get_random_interval(interval: float, min: float, max: float) -> float:
    return interval + random.uniform(min, max)
