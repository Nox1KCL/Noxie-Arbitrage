from parsers.config.config import Config
from pathlib import Path
import pytest


def test_load_success(tmp_path: Path):
    config_dir = tmp_path / "config"
    config_dir.mkdir()
    config_file: Path = config_dir / "config.toml"

    _ = config_file.write_text(
        """
        [scraper]
        workers_num = 3
        user_agent = []

        [http]
        max_retries = 5
        timeout = 10
        interval = 2.0
        min_delay = 0.5
        max_delay = 2.0

        [logger]
        level = "INFO"
        file_path = "logs/parsers/app.log"
        max_size = "10 MB"
        retention = "30 days"
        compression = "zip"
        json_format = true
        """
    )
    config = Config.load(config_file)

    assert config.scraper is not None
    assert config.http is not None
    assert config.logger is not None

def test_load_empty_path(tmp_path: Path):
    config_dir = tmp_path / "config"
    config_dir.mkdir()
    with pytest.raises(IsADirectoryError):
        _ = Config.load(config_dir)

def test_load_wrong_extension(tmp_path: Path):
    config_file = tmp_path / "config.txt"
    _ = config_file.write_text("mock test")

    with pytest.raises(ValueError, match="Unsupported file type"):
        _ = Config.load(config_file)
