FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bots ./cmd/...

FROM gcr.io/distroless/static

COPY --from=builder /bots /bots

ENTRYPOINT ["/bots"]
