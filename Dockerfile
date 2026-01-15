FROM golang:1.24-alpine AS builder

WORKDIR  /build

COPY go.mod go.sum ./

RUN go mod download
COPY . .

RUN go build -ldflags="-s -w" -v -o print3d-order-bot .

FROM alpine

WORKDIR /app

COPY --from=builder /build/print3d-order-bot /app/print3d-order-bot

CMD ["/app/print3d-order-bot"]