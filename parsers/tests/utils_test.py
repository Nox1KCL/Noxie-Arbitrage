import httpx
from parsers.config.config import HttpConfig
from parsers.utils import again_with_retries, get_data, retry_on_rate_limit
import pytest
from pytest_mock import MockerFixture
from unittest.mock import AsyncMock, MagicMock

@pytest.mark.asyncio
async def test_get_data_success():
    desired_data = {
        "symbol": "BTCUSDT",
        "lastPrice": "50000.0",
        "askPrice": "50001.0",
        "bidPrice": "49999.0",
        "volume": "100.5",
    }
    response = httpx.Response(
        status_code=200,
        json=desired_data
    )

    mock_client = MagicMock()
    mock_client.get = AsyncMock(return_value=response)

    result = await get_data("mock_url", mock_client, MagicMock())

    assert result is not None
    assert result.status_code == 200
    assert result.json() == desired_data

@pytest.mark.asyncio
async def test_get_data_timeout():
    mock_client = MagicMock()
    mock_client.get = AsyncMock(side_effect=httpx.TimeoutException(""))

    result = await get_data("mock_url", mock_client, MagicMock())

    assert result is None

@pytest.mark.asyncio
async def test_get_data_404():
    response = httpx.Response(
        status_code=404
    )

    mock_client = MagicMock()
    mock_client.get = AsyncMock(return_value=response)

    result = await get_data("mock_url", mock_client, MagicMock())

    assert result is None

@pytest.mark.asyncio
async def test_again_with_retries_success(mocker: MockerFixture):
    cfg = HttpConfig(
        max_retries = 5,
        timeout = 10,
        interval = 2.0,
        min_delay = 0.5,
        max_delay = 2.0
    )
    response = httpx.Response(
        status_code=200,
    )

    _ = mocker.patch(
        "parsers.utils.asyncio.sleep",
        new_callable=AsyncMock
    )
    mock_client = MagicMock()
    mock_client.get = AsyncMock(return_value=response)

    result = await again_with_retries("mock_url", mock_client, cfg)

    assert result is not None

@pytest.mark.asyncio
async def test_again_with_retries_fail(mocker: MockerFixture):
    cfg = HttpConfig(
        max_retries = 5,
        timeout = 10,
        interval = 2.0,
        min_delay = 0.5,
        max_delay = 2.0
    )
    response = httpx.Response(
        status_code=0,
    )

    _ = mocker.patch(
        "parsers.utils.asyncio.sleep",
        new_callable=AsyncMock
    )
    mock_client = MagicMock()
    mock_client.get = AsyncMock(return_value=response)

    result = await again_with_retries("mock_url", mock_client, cfg)

    assert result is None

@pytest.mark.asyncio
async def test_retry_on_rate_limit_success(mocker: MockerFixture):
    cfg = HttpConfig(
        max_retries = 5,
        timeout = 10,
        interval = 2.0,
        min_delay = 0.5,
        max_delay = 2.0
    )
    response = httpx.Response(
        status_code=200,
    )

    _ = mocker.patch(
        "parsers.utils.asyncio.sleep",
        new_callable=AsyncMock
    )
    mock_client = MagicMock()
    mock_client.get = AsyncMock(return_value=response)

    result = await retry_on_rate_limit("mock_url", mock_client, cfg)

    assert result is not None

@pytest.mark.asyncio
async def test_retry_on_rate_limit_fail(mocker: MockerFixture):
    cfg = HttpConfig(
        max_retries = 5,
        timeout = 10,
        interval = 2.0,
        min_delay = 0.5,
        max_delay = 2.0
    )
    response = httpx.Response(
        status_code=429,
    )

    _ = mocker.patch(
        "parsers.utils.asyncio.sleep",
        new_callable=AsyncMock
    )
    mock_client = MagicMock()
    mock_client.get = AsyncMock(return_value=response)

    result = await retry_on_rate_limit("mock_url", mock_client, cfg)

    assert result is None
