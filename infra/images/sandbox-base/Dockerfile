FROM ubuntu:22.04

RUN apt-get update && apt-get install -y wget curl ca-certificates python3 && \
    ln -sf /usr/bin/python3 /usr/bin/python && \
    rm -rf /var/lib/apt/lists/*

RUN wget https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc \
    && chmod +x runsc && mv runsc /usr/local/bin/

EXPOSE 8080
