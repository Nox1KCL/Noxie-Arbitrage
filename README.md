# Noxie Arbitrage

Asynchronous cross-exchange cryptocurrency arbitrage platform with distributed microservices architecture, real-time spread processing, and Telegram alert delivery.

---

## Tech Stack

- **Go 1.26**: High-throughput aggregation, filtering, gRPC services, Telegram bot long-polling
- **Python 3.13 (`uv`)**: Async market scrapers (Binance, Bybit) with `httpx` and `asyncio`
- **RabbitMQ**: Message broker for ticker ingestion (`amqp091`)
- **gRPC / Protocol Buffers**: Low-latency inter-service communication
- **PostgreSQL / GORM**: User subscription and alert filter persistence
- **OpenTelemetry**: Distributed tracing and metrics instrumentation
- **VictoriaMetrics & Jaeger**: Metrics storage and distributed trace visualization
- **Grafana**: Real-time observability dashboard
- **Kubernetes / Kustomize**: Declarative container orchestration
- **Docker / Docker Compose**: Multi-container containerization

---

## Architecture Overview

```
                      +-------------------+
                      | Binance / Bybit   |
                      +---------+---------+
                                | (HTTP / Async)
                                v
                      +-------------------+
                      |  Python Parsers   |
                      +---------+---------+
                                | (AMQP)
                                v
                      +-------------------+
                      |     RabbitMQ      |
                      +---------+---------+
                                | (AMQP)
                                v
                      +-------------------+
    +---------------->+   Go Processing   +<-----------------+
    |                 +---------+---------+                  |
    | (gRPC: Reload)            | (gRPC: AlertNotification)  | (Postgres: Read)
    |                           v                            |
+---+---------+       +-------------------+                  |
|   Go Bot    |       |    Go Delivery    |                  |
+---+---------+       +---------+---------+                  |
    |                           | (HTTP)                     |
    | (Postgres: Write)         v                            |
    |                 +-------------------+                  |
    +---------------->+   PostgreSQL DB   |                  |
    |                 +-------------------+                  |
    | (Long-poll)                                            |
    v                                                        |
+---------------------+                                      |
|    Telegram API     |<-------------------------------------+
+---------------------+
```

---

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/Nox1KCL/Noxie-Arbitrage.git
cd Noxie-Arbitrage
```

### 2. Configuration Setup

Copy all example configuration files and populate them with your credentials.

#### Root Environment File

```bash
cp .env.example .env
```

Edit `.env`:
- Set `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
- Set `RABBITMQ_DEFAULT_USER`, `RABBITMQ_DEFAULT_PASS`
- Set `GF_SECURITY_ADMIN_USER`, `GF_SECURITY_ADMIN_PASSWORD`
- Set `TELEGRAM_BOT_TOKEN` (your bot token from @BotFather)

#### Backend Go Config

```bash
cp internal/config/config-example.toml internal/config/config.toml
```

#### Python Parsers Config

```bash
cp parsers/src/parsers/config/config-example.toml parsers/src/parsers/config/config.toml
```

---

## Running the Project

### Option A: Docker Compose (Recommended for local evaluation)

Build and run the entire stack (infrastructure + applications) with one command:

```bash
docker compose up --build -d
```

View logs for all services:

```bash
docker compose logs -f
```

Stop the stack:

```bash
docker compose down
```

---

### Option B: Kubernetes / Minikube (Kustomize)

#### 1. Start Minikube

```bash
minikube start
```

#### 2. Deploy via Kustomize

Ensure `.env`, `internal/config/config.toml`, and `parsers/src/parsers/config/config.toml` are configured in the root directory:

```bash
kubectl apply -k .
```

#### 3. Verify Deployment

```bash
kubectl get pods -o wide
kubectl get services
```

#### 4. Tear Down

```bash
kubectl delete -k .
```

---

## Local Development & Testing

### Go Tests (Race Detector enabled)

```bash
go test -v -race ./...
```

### Python Parser Tests

```bash
cd parsers
uv sync
uv run pytest -v
```

---

## Telegram Bot Usage

Interact with the bot using the following commands:

- `/start`: Display welcome message and command overview
- `/subscribe SYMBOL SPREAD% VOLUME CHANGE%`: Subscribe to pair spread alerts
  - Example: `/subscribe BTCUSDT 0.5 100 1.0`
- `/subscriptions`: List your active pair subscriptions
- `/unsubscribe SYMBOL`: Remove subscription for a given pair
  - Example: `/unsubscribe BTCUSDT`

---

## Observability & Service Endpoints

When running via Docker Compose, the following services are available locally:

| Service | Port | Endpoint / Description |
| :--- | :--- | :--- |
| **Grafana** | `3000` | `http://localhost:3000` (Metrics dashboard) |
| **Jaeger UI** | `16686` | `http://localhost:16686` (Distributed traces) |
| **VictoriaMetrics** | `8428` | `http://localhost:8428` (Prometheus/OTel metrics backend) |
| **RabbitMQ Management** | `15672` | `http://localhost:15672` (Broker UI) |
| **PostgreSQL** | `5432` | Database instance |
| **Processing gRPC** | `50050` | Inter-service processing gRPC server |
| **Delivery gRPC** | `50051` | Alert delivery gRPC server |
