import time
from typing import override

import httpx

from parsers.broker import Broker
from parsers.config.config import HttpConfig
from parsers.scrapers.base import BasicScraper
from parsers.scrapers.models import TickerForm
from parsers.utils import get_data


class BinanceScraper(BasicScraper):
    def __init__(self, client: httpx.AsyncClient, broker: Broker, api_url:str):
        super().__init__(client, broker, api_url)

    @override
    async def fetch_data(self, symbol: str, cfg: HttpConfig) -> TickerForm:
        target_url = f"{self.api_url}{symbol}"
        response = await get_data(target_url, self.client, cfg)
        raw_data = response.json()

        try:
            return TickerForm(
                exchangeName="Binance",
                symbol=raw_data["symbol"],
                currentPrice=float(raw_data["lastPrice"]),
                bestAsk=float(raw_data["askPrice"]),
                bestBid=float(raw_data["bidPrice"]),
                volume=float(raw_data["volume"]),
                timestamp=int(time.time() * 1000),
            )
        except KeyError as e:
            raise Exception(f"Missing key: {e}")
