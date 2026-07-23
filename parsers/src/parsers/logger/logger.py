import sys
from loguru import logger
from parsers.config import LoggerConfig

def setup_logger(cfg: LoggerConfig) -> None:
    logger.remove()

    _ = logger.add(
            sys.stdout,
            level=cfg.level,
            colorize=True,
            format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level:10}</level> | <cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - <level>{message}</level>",
    )

    _ = logger.add(
        cfg.logs_dir,
        level=cfg.level,
        rotation=cfg.max_size,
        retention=cfg.retention,
        compression=cfg.compression,
        serialize=cfg.json_format,
        encoding="utf-8",
        enqueue=True, # Для асинхронності
    )