# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /submission ./cmd/...

# ---- Runtime stage ----
FROM gcr.io/distroless/static

COPY --from=builder /submission /submission

EXPOSE 8081

ENTRYPOINT ["/submission"]
