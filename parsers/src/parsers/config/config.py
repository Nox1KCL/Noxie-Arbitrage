import tomllib
from pathlib import Path

from pydantic import BaseModel, Field

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent.parent
LOGS_DIR = str(PROJECT_ROOT / "logs" / "app.log")

class ScraperConfig(BaseModel):
    workers_num: int
    user_agent: list[str]

class HttpConfig(BaseModel):
    max_retries: int
    timeout: float
    interval: float
    min_delay: float = Field(default=0.5)
    max_delay: float = Field(default=2)

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
    def load(cls, path: str | Path | None = None) -> "Config":
        if path is None:
             path = Path(__file__).parent / "config.toml"
        config_path = Path(path)
        if not config_path.exists():
            raise FileNotFoundError(f"Config file not found: {config_path}")

        if not config_path.is_file():
            raise IsADirectoryError(f"Current path leads to directory: {config_path}")

        if config_path.suffix != ".toml":
            raise ValueError(f"Unsupported file type: {config_path.suffix}")
        
        with config_path.open("rb") as f:
            data = tomllib.load(f)

        return cls.model_validate(data)
