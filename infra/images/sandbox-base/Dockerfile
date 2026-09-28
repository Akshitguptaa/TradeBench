FROM ubuntu:22.04

RUN apt-get update && apt-get install -y wget curl ca-certificates python3 gnupg && \
    ln -sf /usr/bin/python3 /usr/bin/python && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://gvisor.dev/archive.key | gpg --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" | tee /etc/apt/sources.list.d/gvisor.list > /dev/null && \
    apt-get update && apt-get install -y runsc && \
    rm -rf /var/lib/apt/lists/*

EXPOSE 8080

