FROM golang:1.23.3-bookworm

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    openssh-client \
    ; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

ENV PATH="/app/docker_files:${PATH}"

RUN go mod download && go mod verify
RUN go build -o server_binary ./server

RUN mkdir -p /var/log/laforge /root/.ssh && \
    ssh-keyscan github.com >> /root/.ssh/known_hosts

EXPOSE 8080 50051

CMD ["./server_binary"]
