FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /sandbox ./cmd/...

FROM gcr.io/distroless/static

COPY --from=builder /sandbox /sandbox

EXPOSE 8082

ENTRYPOINT ["/sandbox"]
