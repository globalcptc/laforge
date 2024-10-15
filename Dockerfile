FROM golang:1.21-bullseye

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    git \
    xz-utils\
    ; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /
ADD https://ziglang.org/builds/zig-linux-x86_64-0.14.0-dev.1860+2e2927735.tar.xz /zig.tar.xz
RUN mkdir /zig
RUN tar -xvf /zig.tar.xz -C /zig --strip-components=1
ENV PATH="/zig:${PATH}"

# ADD https://github.com/mattnite/gyro/releases/download/0.5.0/gyro-0.5.0-linux-x86_64.tar.gz /gyro-0.5.0-linux-x86_64.tar.gz
# RUN tar -xvf /gyro-0.5.0-linux-x86_64.tar.gz
# ENV PATH="/gyro-0.5.0-linux-x86_64/bin:${PATH}"

RUN mkdir /app
ADD . /app/
WORKDIR /app

ENV PATH="/app/docker_files:${PATH}"

RUN go mod download && go mod verify
RUN go build -o server_binary server/server.go

RUN mkdir /var/log/laforge

RUN mkdir ~/.ssh
RUN ssh-keygen -F github.com || ssh-keyscan github.com >> ~/.ssh/known_hosts

EXPOSE 8080 50051

CMD ["./server_binary"]
