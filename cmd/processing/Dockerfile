FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
ADD go.mod go.sum ./
RUN go mod download
ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/nxe-processing ./cmd/processing/main.go

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/ .

CMD ["./nxe-processing"]
