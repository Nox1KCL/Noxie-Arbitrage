from pydantic import BaseModel, Field


class TickerForm(BaseModel):
    exchange_name: str = Field(alias="exchangeName")
    symbol: str
    current_price: float = Field(ge=0, alias="currentPrice")
    best_ask: float = Field(ge=0, alias="bestAsk")
    best_bid: float = Field(ge=0, alias="bestBid")
    volume: float = Field(ge=0)
    timestamp: int = Field(ge=0)