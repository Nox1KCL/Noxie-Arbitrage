import asyncio

import httpx
from loguru import logger

from parsers.broker import Broker
from parsers.config.config import Config, HttpConfig
from parsers.logger.logger import setup_logger
from parsers.scrapers.binance import BinanceScraper
from parsers.utils import get_env, get_random_interval, get_url


async def worker(worker_id: int, queue: asyncio.Queue[str], scraper: BinanceScraper, cfg: HttpConfig):
    logger.info(f"worker {worker_id} starting")
    while True:
        symbol = await queue.get()
        await scraper.process_single(symbol, cfg)
        queue.task_done()

async def dispatcher(symbols: list[str], queue: asyncio.Queue[str], interval: float):
    while True:
        logger.info("dispatching symbols")
        for s in symbols:
            await queue.put(s)

        await asyncio.sleep(interval)


async def main():
    get_env()
    cfg = Config.load()
    setup_logger(cfg.logger)

    broker = Broker()
    await broker.connect(get_url())
    await broker.make_queue("parser.binance.ticker")

    try:
        ticker_list = ["BTCUSDT"]
        queue = asyncio.Queue[str]()

        async with httpx.AsyncClient() as client:
            binance = BinanceScraper(
                client=client,
                broker=broker,
                api_url="https://api.binance.com/api/v3/ticker/24hr?symbol="
            )

            interval = get_random_interval(cfg.http.interval, cfg.http.min_delay, cfg.http.max_delay)
            async with asyncio.TaskGroup() as tg:
                _ = tg.create_task(dispatcher(ticker_list, queue, interval))
                for i in range(cfg.scraper.workers_num):
                    _ = tg.create_task(worker(i, queue, binance, cfg.http))
    finally:
        await broker.tidy()

if __name__ == "__main__":
    asyncio.run(main())
