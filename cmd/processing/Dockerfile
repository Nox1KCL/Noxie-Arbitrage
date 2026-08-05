FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/nxe-processing ./cmd/processing

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/ .

CMD ["./nxe-processing"]
