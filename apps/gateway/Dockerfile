FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /gateway ./cmd/...

FROM gcr.io/distroless/static

COPY --from=builder /gateway /gateway

EXPOSE 8080

ENTRYPOINT ["/gateway"]
