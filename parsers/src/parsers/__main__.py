import asyncio
import time
from opentelemetry import metrics

import httpx, signal
from loguru import logger

from parsers.broker import Broker
from parsers.config.config import Config, HttpConfig
from parsers.logger.logger import setup_logger
from parsers.scrapers.base import BasicScraper
from parsers.scrapers.binance import BinanceScraper
from parsers.scrapers.bybit import BybitScraper
from parsers.utils import get_env, get_random_interval, get_url

meter = metrics.get_meter(__name__)

counter = meter.create_counter(
    "counter",
    description="Counting of processed valuables"
)
histogram = meter.create_histogram(
    "histogram",
    description="Calculating total time of processes"
)


async def worker(worker_id: int, queue: asyncio.Queue[str], scraper: BasicScraper, cfg: HttpConfig, stop_event: asyncio.Event):
    logger.info(f"worker {worker_id} starting")
    exchange = type(scraper).__name__

    while not stop_event.is_set():
        try:
            symbol = await asyncio.wait_for(queue.get(), timeout=1.0)
        except asyncio.TimeoutError:
            continue

        start = time.time()

        try:
            await scraper.process_single(symbol, cfg)

            counter.add(1, {"exchange": exchange, "stage": "worker", "type": "scraper.processed.success"})

        except Exception as e:
            counter.add(1, {"exchange": exchange, "stage": "worker", "type": "scraper.errors"})
            logger.error(f"Error parsing {symbol}: {e}")

        finally:
            duration = time.time() - start
            histogram.record(duration, {"exchange": exchange, "stage": "worker", "type": "spent.time"})

            queue.task_done()

async def dispatcher(symbols: list[str], queues: list[asyncio.Queue[str]], interval: float, stop_event: asyncio.Event):
    while not stop_event.is_set():
        logger.info("dispatching symbols")
        for s in symbols:
            for q in queues:
                await q.put(s)

        await asyncio.sleep(interval)


async def main():
    get_env()
    cfg = Config.load()
    setup_logger(cfg.logger)

    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop_event.set)
        
    logger.info("starting initialize broker")
    broker = Broker()
    await broker.connect(get_url())
    await broker.make_queue("parser.binance.ticker")

    ticker_list = ["BTCUSDT"]
    binance_queue = asyncio.Queue[str]()
    bybit_queue = asyncio.Queue[str]()

    logger.info("starting httpx client session")
    async with httpx.AsyncClient() as client:
        binance = BinanceScraper(
            client=client,
            broker=broker,
            api_url="https://api.binance.com/api/v3/ticker/24hr?symbol="
        )
        bybit = BybitScraper(
            client=client,
            broker=broker,
            api_url="https://api.bybit.com/v5/market/tickers?category=spot&symbol="
        )
        queues = [binance_queue, bybit_queue]
        batch_tasks: list[asyncio.Task[None]] = []
        interval = get_random_interval(cfg.http.interval, cfg.http.min_delay, cfg.http.max_delay)

        dispatcher_task = asyncio.create_task(dispatcher(ticker_list, queues, interval, stop_event))

        batch_tasks.append(dispatcher_task)
        for i in range(cfg.scraper.workers_num):
            task= asyncio.create_task(worker(i, binance_queue, binance, cfg.http, stop_event))
            batch_tasks.append(task)

        for i in range(cfg.scraper.workers_num):
            task = asyncio.create_task(worker(i, bybit_queue, bybit, cfg.http, stop_event))
            batch_tasks.append(task)

        _ = await stop_event.wait()

        logger.warning("Starting gracefully shutdown..")
        _ = await asyncio.gather(*batch_tasks, return_exceptions=True)
        await broker.tidy()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass
