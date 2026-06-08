FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /scorer ./cmd/...

FROM gcr.io/distroless/static

COPY --from=builder /scorer /scorer

ENTRYPOINT ["/scorer"]
