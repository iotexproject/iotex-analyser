# syntax=docker/dockerfile:1.4
FROM golang:1.23-bullseye AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    make gcc git libc-dev build-essential \
 && rm -rf /var/lib/apt/lists/*

# Copy go.mod / go.sum first so dependency download can be cached separately
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Parallel build with BuildKit cache mounts for go modules and build cache.
# `make all` recursively invokes `make -j$(nproc) plugins` to parallelise the
# per-plugin go build invocations declared in the Makefile.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make all

FROM golang:1.23-bullseye

WORKDIR /app
COPY --from=builder /app/*.so /app/
COPY --from=builder /app/config.yml /app/
COPY --from=builder /app/iotex-analyser /app/
ENTRYPOINT ["./iotex-analyser"]
