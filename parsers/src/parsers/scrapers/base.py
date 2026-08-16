from abc import ABC, abstractmethod
from opentelemetry import trace

import httpx
from loguru import logger

from parsers.broker import Broker
from parsers.config.config import HttpConfig
from parsers.scrapers.models import TickerForm

tracer = trace.get_tracer(__name__)

class BasicScraper(ABC):
    api_url: str
    client: httpx.AsyncClient
    broker: Broker

    def __init__(self, client: httpx.AsyncClient, broker: Broker, api_url: str):
        self.client = client
        self.broker = broker
        self.api_url = api_url


    @abstractmethod
    async def fetch_data(self, symbol: str, cfg: HttpConfig) -> TickerForm:
        pass
    
    @tracer.start_as_current_span("process_single_ticker")
    async def process_single(self, symbol: str, cfg: HttpConfig):
        try:
            span = trace.get_current_span()
            span.set_attribute("ticker.symbol", symbol)

            data: TickerForm = await self.fetch_data(symbol, cfg)

            binary_data: bytes = data.model_dump_json(by_alias=True).encode("utf-8")
            
            logger.info(f"Send message to broker for symbol {symbol}")
            await self.broker.load(binary_data)

        except Exception as e:
            logger.exception(f"Fetching data from arbitrage {e}")
