FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR  /build

COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN go build -ldflags="-s -w" -v -o print3d-order-bot .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/print3d-order-bot /app/print3d-order-bot

CMD ["/app/print3d-order-bot"]