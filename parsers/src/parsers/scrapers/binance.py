from typing import override

from parsers.scrapers.models import TickerForm
from parsers.scrapers.base import BasicScraper
from parsers.broker import Broker
import httpx

class BinanceScraper(BasicScraper):
    def __init__(self, client: httpx.AsyncClient, broker: Broker, api_url:str):
        super().__init__(client, broker, api_url)

    @override
    async def fetch_data(self, symbol: str) -> TickerForm:
        response = await self.client.get(self.api_url)
        if response.status_code != 200:
            # TODO: створити власний ексепшн
            raise Exception(f"Failed to fetch data: {response.status_code}")
        data = response.json()
        return TickerForm(**data)