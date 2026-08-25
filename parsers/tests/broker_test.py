import pytest
from unittest.mock import AsyncMock, MagicMock
from parsers.broker import Broker


@pytest.mark.asyncio
async def test_load_success():
    broker = Broker()
    broker.channel = MagicMock()
    broker.channel.default_exchange.publish = AsyncMock()

    await broker.load(b"test_data")

    broker.channel.default_exchange.publish.assert_called_once()


@pytest.mark.asyncio
async def test_load_without_connect():
    broker = Broker()

    with pytest.raises(RuntimeError, match="connect"):
        await broker.load(b"test_data")


@pytest.mark.asyncio
async def test_make_queue_without_connect():
    broker = Broker()

    with pytest.raises(RuntimeError, match="connect"):
        await broker.make_queue("test_queue")


@pytest.mark.asyncio
async def test_unload_without_connect():
    broker = Broker()

    with pytest.raises(RuntimeError, match="connect"):
        await broker.unload()
