FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /dummy-orderbook ./cmd/main.go

FROM gcr.io/distroless/static:latest

COPY --from=builder /dummy-orderbook /dummy-orderbook

EXPOSE 8080

ENTRYPOINT ["/dummy-orderbook"]
