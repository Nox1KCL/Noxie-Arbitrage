import sys
from pathlib import Path
from venv import logger
from pydantic import BaseModel, Field, PostgresDsn

if sys.version_info >= (3, 11):
    import tomllib
else:
    import tomli as tomllib

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent.parent.parent
LOGS_DIR = str("logs" / PROJECT_ROOT / "app.log")

class ScraperConfig(BaseModel):
    workers_num: int = Field()
    cycle_period_seconds: float = Field()
    user_agent: list[str] = Field()

class HttpConfig(BaseModel):
    max_retries: int = Field()
    timeout_seconds: float = Field()
    min_delay_seconds: float = Field(default=0.5)
    max_delay_seconds: float = Field(default=2)

class LoggerConfig(BaseModel):
    level: str = "INFO"
    logs_dir: str = LOGS_DIR
    max_size: str = Field(default="10 MB")
    retention: str = Field(default="30 days")
    compression: str = Field(default="zip")
    json_format: bool = Field(default=True)

class Config(BaseModel):
    scraper: ScraperConfig
    http: HttpConfig
    logger: LoggerConfig

    @classmethod
    def load(cls, path: str | Path = "config.toml") -> "Config":
        config_path = Path(path)
        if not config_path.exists():
            raise FileNotFoundError(f"Config file not found: {config_path}")

        with config_path.open("rb") as f:
            data = tomllib.load(f)

        return cls.model_validate(data)
