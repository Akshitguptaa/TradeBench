FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /ingester ./cmd/...

FROM gcr.io/distroless/static

COPY --from=builder /ingester /ingester

ENTRYPOINT ["/ingester"]
