# One Dockerfile, one build target per real service (docker-compose.yml
# picks the target with `build.target`) -- "Everything is a container,
# and nothing depends on the platform underneath... A Compose file that
# actually works is part of the deliverable." cmd/laforge-lsp and cmd/laforge-agent-factory
# aren't here: neither is a long-running service (an editor launches the
# first as a subprocess; the second is invoked on demand, not something
# docker-compose runs continuously).
#
# CGO_ENABLED=0: every real dependency here (pgx, the standard library)
# is pure Go, so a static binary needs nothing from the final image's
# libc at all -- that's what makes the distroless final stages below
# possible.

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
ENV CGO_ENABLED=0
RUN go build -o /out/laforge-api ./cmd/laforge-api
RUN go build -o /out/laforge-orchestrator ./cmd/laforge-orchestrator
RUN go build -o /out/laforge-gateway ./cmd/laforge-gateway
RUN go build -o /out/laforge-runner ./cmd/laforge-runner
RUN go build -o /out/laforge-migrate ./cmd/laforge-migrate

# WORKDIR / is explicit on every stage below, deliberately: the
# distroless nonroot base image's own default is /home/nonroot, not /,
# which is exactly the kind of implicit, base-image-specific detail that
# silently breaks a relative path (found for real: GITHUB_APP_PRIVATE_KEY_PATH
# resolving against the wrong directory, a relative path resolving fine
# for `go run` from the repo root but not under compose, until this was
# pinned down explicitly). Every relative path this project's own .env
# uses should resolve the same way in both places.

# Services that resolve real repositories shell out to `git` (internal/checkout),
# so they can't run on distroless/static. A slim Debian with git + CA certs is
# the smallest base that actually works for them.
FROM debian:bookworm-slim AS gitbase
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

FROM gitbase AS api
WORKDIR /
COPY --from=build /out/laforge-api /usr/local/bin/laforge-api
ENTRYPOINT ["/usr/local/bin/laforge-api"]

FROM gitbase AS orchestrator
WORKDIR /
COPY --from=build /out/laforge-orchestrator /usr/local/bin/laforge-orchestrator
ENTRYPOINT ["/usr/local/bin/laforge-orchestrator"]

FROM gcr.io/distroless/static-debian12:nonroot AS gateway
WORKDIR /
COPY --from=build /out/laforge-gateway /usr/local/bin/laforge-gateway
ENTRYPOINT ["/usr/local/bin/laforge-gateway"]

# Agent base binaries: cross-compile the Rust on-host agent once per platform.
# The runner patches a per-host COPY of these at deploy time (mTLS certs +
# gateway address) and seals the self-hash itself (internal/agentdelivery), so
# these need no post-build step. Building them here means `docker compose up
# --build` produces them into the runner image -- they are build artifacts,
# never committed, and never fetched from a git push. This stage only rebuilds
# when agent/ changes, and Rust never lands in the final runner image (only the
# two binaries are copied out below).
FROM rust:1-bookworm AS agent-build
# Cross-compiling the agent (which pulls in ring's C code for TLS) to musl and
# Windows needs a working C cross-toolchain for each target. Debian's musl-gcc
# wrapper can't handle ring's compiler flags, so use cargo-zigbuild: zig is a
# drop-in C cross-compiler for every target, with no per-target gcc/mingw to
# install and wire up.
RUN apt-get update && apt-get install -y --no-install-recommends xz-utils ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && rustup target add x86_64-unknown-linux-musl x86_64-pc-windows-gnu \
 && cargo install --locked cargo-zigbuild \
 && ZIG_VER=0.13.0 && ARCH="$(uname -m)" \
 && curl -fsSL "https://ziglang.org/download/${ZIG_VER}/zig-linux-${ARCH}-${ZIG_VER}.tar.xz" -o /tmp/zig.tar.xz \
 && mkdir -p /opt/zig && tar -xJf /tmp/zig.tar.xz -C /opt/zig --strip-components=1 \
 && rm /tmp/zig.tar.xz && ln -s /opt/zig/zig /usr/local/bin/zig
WORKDIR /agent
COPY agent/ .
# zigbuild supplies the linker for both targets; drop the repo's own gcc/mingw
# linker config so it isn't picked up instead of zig's.
RUN rm -f .cargo/config.toml .cargo/config \
 && cargo zigbuild --release --target x86_64-unknown-linux-musl \
 && cargo zigbuild --release --target x86_64-pc-windows-gnu \
 && mkdir -p /agent-base \
 && cp target/x86_64-unknown-linux-musl/release/laforge-agent /agent-base/agent-x86_64-unknown-linux-musl \
 && cp target/x86_64-pc-windows-gnu/release/laforge-agent.exe /agent-base/agent-x86_64-pc-windows-gnu.exe

FROM gitbase AS runner
WORKDIR /
COPY --from=build /out/laforge-runner /usr/local/bin/laforge-runner
COPY --from=agent-build /agent-base/ /agent-base/
ENV AGENT_BASE_DIR=/agent-base
ENTRYPOINT ["/usr/local/bin/laforge-runner"]

# migrate runs cmd/laforge-migrate (goose used as a library, Postgres
# only) rather than `go install`ing goose's own CLI: the CLI
# blank-imports a driver for every database goose supports (MySQL,
# ClickHouse, libsql, YDB, MSSQL, SQLite, ...) so it can dispatch on a
# dialect flag, which drags in the Azure SDK, gRPC, and OpenTelemetry
# transitively -- a real, measured difference found while building this
# image, not a guess (see cmd/laforge-migrate/main.go's own doc
# comment). distroless like every other stage: this binary needs nothing
# from a real OS image either.
FROM gcr.io/distroless/static-debian12:nonroot AS migrate
WORKDIR /
COPY --from=build /out/laforge-migrate /usr/local/bin/laforge-migrate
COPY migrations /migrations
ENV MIGRATIONS_DIR=/migrations
ENTRYPOINT ["/usr/local/bin/laforge-migrate"]
