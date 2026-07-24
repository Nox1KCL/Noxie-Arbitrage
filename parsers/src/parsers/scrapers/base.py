from parsers.scrapers.models import TickerForm
from abc import ABC
import httpx
import asyncio
from parsers.broker import Broker
from abc import abstractmethod
from loguru import logger


class BasicScraper(ABC):
    api_url: str
    client: httpx.AsyncClient
    broker: Broker

    def __init__(self, client: httpx.AsyncClient, broker: Broker, api_url: str):
        self.client = client
        self.broker = broker
        self.api_url = api_url


    @abstractmethod
    async def fetch_data(self, symbol: str) -> TickerForm:
        pass

    async def run(self, symbol: str, interval: int):
        while True:
            try:
                data: TickerForm = await self.fetch_data(symbol)
                binary_data: bytes = data.model_dump_json(by_alias=True).encode("utf-8")
                await self.broker.load(binary_data)

            except Exception as e:
                logger.exception(f"Fetching data from arbitrage {e}")

            await asyncio.sleep(interval)
