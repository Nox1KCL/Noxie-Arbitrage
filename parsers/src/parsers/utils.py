from dotenv import load_dotenv
import os
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent.parent

def get_env():
    _ = load_dotenv(PROJECT_ROOT / ".env")
    return

def get_url() -> str:
    host = os.getenv("RABBIT_HOST")
    pwd = os.getenv("RABBIT_PWD")
    user = os.getenv("RABBIT_USER")
    port = os.getenv("RABBIT_PORT")

    url = f"amqp://{user}:{pwd}@{host}:{port}/"
    return url
