import time
from typing import override

import httpx

from loguru import logger
from parsers.config.config import HttpConfig
from parsers.scrapers.base import BasicScraper
from parsers.scrapers.models import TickerForm
from parsers.utils import get_data


class BinanceScraper(BasicScraper):
    def __init__(self, client: httpx.AsyncClient, api_url:str):
        super().__init__(client, api_url)

    @override
    async def fetch_data(self, symbol: str, cfg: HttpConfig) -> TickerForm | None:
        target_url = f"{self.api_url}{symbol}"
        response = await get_data(target_url, self.client, cfg)
        if response is None:
            return None

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
            logger.error(f"Missing key: {e}")
            return None
