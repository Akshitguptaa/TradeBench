FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

RUN git clone --depth 1 https://github.com/minio/minio.git /src/minio
WORKDIR /src/minio
RUN CGO_ENABLED=0 go build -o /usr/local/bin/minio .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /usr/local/bin/minio /usr/local/bin/minio

EXPOSE 9000 9001

ENTRYPOINT ["minio"]
