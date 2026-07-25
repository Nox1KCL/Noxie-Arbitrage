from dataclasses import dataclass
import aio_pika
from aio_pika.abc import AbstractChannel, AbstractQueue

@dataclass
class Broker:
    connection: aio_pika.Connection | None = None
    channel: AbstractChannel | None = None
    queue_name: str = ""

    async def load(self, body: bytes):
        if self.channel is None:
            raise RuntimeError("connect() first")

        _ = await self.channel.default_exchange.publish(
            aio_pika.Message(
                body,
                delivery_mode=aio_pika.DeliveryMode.PERSISTENT
            ),
            routing_key=self.queue_name,
        )

    async def unload(self) -> AbstractQueue | None:
        if self.channel is None:
            raise RuntimeError("connect() first")

        try:
            return await self.channel.get_queue(
                self.queue_name,
                ensure=True
            )
        except aio_pika.exceptions.ChannelNotFoundEntity:
            return None

    async def make_queue(self, queue_name: str):
        if self.channel is None:
            raise RuntimeError("connect() first")

        self.queue_name = queue_name
        _ = await self.channel.declare_queue(
            queue_name,
            durable=True
        )

    async def connect(self, url: str):
        self.connection = await aio_pika.connect_robust(url)
        self.channel = await self.connection.channel()

    async def tidy(self):
        if self.channel:
            await self.channel.close()
        if self.connection:
            await self.connection.close()
