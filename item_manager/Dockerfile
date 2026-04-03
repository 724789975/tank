FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o item_manager main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/item_manager .
COPY etc/server-dev.yaml etc/server-dev.yaml

EXPOSE 8084 8884

CMD ["./item_manager"]