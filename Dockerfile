FROM golang:alpine AS builder

WORKDIR  /build

ADD go.mod .

RUN go mod download
COPY . .

RUN go build -ldflags="-s -w" -o print3d-order-bot .

FROM alpine

WORKDIR /app

COPY --from=builder /build/print3d-order-bot /app/print3d-order-bot

CMD ["/app/print3d-order-bot"]