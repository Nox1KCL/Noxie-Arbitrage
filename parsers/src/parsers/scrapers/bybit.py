from typing import override

from loguru import logger
from parsers.scrapers.base import BasicScraper
import httpx, time
from parsers.scrapers.models import TickerForm
from parsers.utils import HttpConfig, get_data

class BybitScraper(BasicScraper):
    def __init__(self, client: httpx.AsyncClient, api_url: str):
        super().__init__(client, api_url)

    @override
    async def fetch_data(self, symbol: str, cfg: HttpConfig) -> TickerForm | None:
        target_url = f"{self.api_url}{symbol}"
        response = await get_data(target_url, self.client, cfg)
        if response is None:
            return None

        raw_data = response.json()
        prepared_data = raw_data["result"]["list"][0]

        try:
            return TickerForm(
                exchangeName="Bybit",
                symbol=prepared_data["symbol"],
                currentPrice=float(prepared_data["lastPrice"]),
                bestAsk=float(prepared_data["ask1Price"]),
                bestBid=float(prepared_data["bid1Price"]),
                volume=float(prepared_data["volume24h"]),
                timestamp=int(time.time() * 1000),
            )
        except KeyError as e:
            logger.error(f"Missing key: {e}")
            return None
