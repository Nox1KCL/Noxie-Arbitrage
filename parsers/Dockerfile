FROM python:3.13-slim AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
  PYTHONUNBUFFERED=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  libpq-dev \
  && rm -rf /var/lib/apt/lists/*

RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

COPY requirements.txt pyproject.toml ./
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.13-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
  PYTHONUNBUFFERED=1 \
  PATH="/opt/venv/bin:$PATH"

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
  libpq5 \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /opt/venv /opt/venv

RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

COPY --chown=appuser:appuser . .

ENV PYTHONPATH=/app/src
CMD ["opentelemetry-instrument", "python", "-m", "parsers"]
