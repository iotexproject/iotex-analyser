# syntax=docker/dockerfile:1.4
FROM golang:1.23-alpine AS builder
WORKDIR /app

RUN apk add --no-cache make gcc git musl-dev

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

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/*.so /app/
COPY --from=builder /app/config.yml /app/
COPY --from=builder /app/iotex-analyser /app/
ENTRYPOINT ["./iotex-analyser"]
