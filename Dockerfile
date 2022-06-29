FROM golang:1.18.3-bullseye

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    git \
    xz-utils\
    ; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /
ADD https://ziglang.org/download/0.9.1/zig-linux-x86_64-0.9.1.tar.xz /zig-linux-x86_64-0.9.1.tar.xz
RUN tar -xvf /zig-linux-x86_64-0.9.1.tar.xz
ENV PATH="/zig-linux-x86_64-0.9.1:${PATH}"

ADD https://github.com/mattnite/gyro/releases/download/0.5.0/gyro-0.5.0-linux-x86_64.tar.gz /gyro-0.5.0-linux-x86_64.tar.gz
RUN tar -xvf /gyro-0.5.0-linux-x86_64.tar.gz
ENV PATH="/gyro-0.5.0-linux-x86_64/bin:${PATH}"

RUN mkdir /app 
ADD . /app/
WORKDIR /app 

ENV PATH="/app/docker_files:${PATH}"

RUN go mod download && go mod verify
RUN go build -o server_binary server/server.go

RUN mkdir /var/log/laforge

EXPOSE 8080 50051

CMD ["./server_binary"]