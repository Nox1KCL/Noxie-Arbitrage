import httpx
from parsers.scrapers.binance import BinanceScraper
from parsers.scrapers.bybit import BybitScraper
import pytest
from pytest_mock import MockerFixture
from unittest.mock import AsyncMock, MagicMock
from parsers.scrapers.models import TickerForm

@pytest.mark.asyncio
async def test_process_single(mocker: MockerFixture):
    ticker = TickerForm(
        exchangeName="binance",
        symbol="BTCUSDT",
        currentPrice=50000,
        bestAsk=50000,
        bestBid=50000,
        volume=1000,
        timestamp=0,
    )
    scraper = BinanceScraper(client=MagicMock(), api_url="mock_url")
    _ = mocker.patch.object(
        scraper,
        'fetch_data',
        newcallable=AsyncMock,
        return_value=ticker
    )
    data = await scraper.process_single("BTCUSDT", MagicMock())

    assert data is not None
    assert b"BTCUSDT" in data
    assert b"50000" in data

@pytest.mark.asyncio
async def test_fetch_data_Binance(mocker: MockerFixture):
    binance = BinanceScraper(
        client=MagicMock(),
        api_url="mock_url"
    )
    response = httpx.Response(
        status_code=200,
        json={
          "symbol": "BTCUSDT",
          "priceChange": "-645.13000000",
          "priceChangePercent": "-0.808",
          "weightedAvgPrice": "79594.15327088",
          "prevClosePrice": "79880.94000000",
          "lastPrice": "79235.80000000",
          "lastQty": "0.00026000",
          "bidPrice": "79235.79000000",
          "bidQty": "3.92442000",
          "askPrice": "79235.80000000",
          "askQty": "1.10829000",
          "openPrice": "79880.93000000",
          "highPrice": "81272.62000000",
          "lowPrice": "78120.74000000",
          "volume": "28526.61472000",
          "quoteVolume": "2270551744.32291180",
          "openTime": 1787585392001,
          "closeTime": 1787671792001,
          "firstId": 6611358132,
          "lastId": 6618114206,
          "count": 6756075
        }
    )

    _ = mocker.patch(
        'parsers.scrapers.binance.get_data',
        new_callable=AsyncMock,
        return_value=response
    )
    ticker = await binance.fetch_data("BTCUSDT", MagicMock())

    assert ticker is not None
    assert ticker.exchange_name == "Binance"
    assert ticker.symbol == "BTCUSDT"
    assert ticker.current_price == 79235.80
    assert ticker.best_ask == 79235.80
    assert ticker.best_bid == 79235.79
    assert ticker.volume == 28526.61472

@pytest.mark.asyncio
async def test_fetch_data_Bybit(mocker: MockerFixture):
    bybit = BybitScraper(
        client=MagicMock(),
        api_url="mock_url"
    )
    response = httpx.Response(
        status_code=200,
        json={
            "retCode": 0,
            "retMsg": "OK",
            "result": {
                "category": "spot",
                "list": [
                    {
                        "symbol": "BTCUSDT",
                        "bid1Price": "50000.10",
                        "bid1Size": "1.5",
                        "ask1Price": "50000.20",
                        "ask1Size": "2.1",
                        "lastPrice": "50000.15",
                        "prevPrice24h": "49000.00",
                        "price24hPcnt": "0.0204",
                        "highPrice24h": "51000.00",
                        "lowPrice24h": "48500.00",
                        "turnover24h": "1000000000.00",
                        "volume24h": "20000.55"
                    }
                ]
            },
            "retExtInfo": {},
            "time": 1787585392001
        }
    )

    _ = mocker.patch(
        'parsers.scrapers.bybit.get_data',
        new_callable=AsyncMock,
        return_value=response
    )

    ticker = await bybit.fetch_data("BTCUSDT", MagicMock())

    assert ticker is not None
    assert ticker.exchange_name == "Bybit"
    assert ticker.symbol == "BTCUSDT"
    assert ticker.current_price == 50000.15
    assert ticker.best_ask == 50000.20
    assert ticker.best_bid == 50000.10
    assert ticker.volume == 20000.55
