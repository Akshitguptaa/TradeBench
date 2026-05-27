FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /leaderboard ./cmd/...

FROM gcr.io/distroless/static

COPY --from=builder /leaderboard /leaderboard

ENTRYPOINT ["/leaderboard"]
