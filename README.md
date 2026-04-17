# IoTeX Analyser

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/iotexproject/iotex-analyser)](https://goreportcard.com/report/github.com/iotexproject/iotex-analyser)
[![Latest Release](https://img.shields.io/github/v/release/iotexproject/iotex-analyser)](https://github.com/iotexproject/iotex-analyser/releases)
[![Docker Image](https://img.shields.io/badge/ghcr.io-iotexproject%2Fiotex--analyser-blue?logo=docker)](https://github.com/iotexproject/iotex-analyser/pkgs/container/iotex-analyser)

**IoTeX Analyser** indexes data from the [IoTeX blockchain](https://iotex.io) into the database of your choice (PostgreSQL, MySQL, SQLite, ClickHouse) and exposes it for downstream querying — including a managed GraphQL endpoint via Hasura.

It is built around a **dynamic plugin system**: indexing logic ships as Go plugins (`.so` files) that can be loaded and unloaded at runtime, with no service restart. A growing library of [58+ built-in plugins](plugins/) covers common needs (blocks, actions, ERC20/721/1155, staking, rewards, gas tracking, account income, …), and you can drop in your own.

![Technical architecture](docs/assets/images/162696321861.png)

---

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Building from Source](#building-from-source)
- [Running the Server](#running-the-server)
- [Plugin System](#plugin-system)
  - [Writing a Plugin](#writing-a-plugin)
  - [Loading and Unloading at Runtime](#loading-and-unloading-at-runtime)
- [GraphQL API](#graphql-api)
- [Releases](#releases)
- [Contributing](#contributing)
- [License](#license)

---

## Quick Start

The fastest way to try Analyser is via Docker Compose, which provisions both Postgres and the Analyser service:

```sh
docker-compose up
```

To use a prebuilt image directly:

```sh
docker pull ghcr.io/iotexproject/iotex-analyser:latest
```

Available image tags:
- `:latest` — latest tagged release (recommended)
- `:vX.Y.Z` — pinned release version
- `:main` — bleeding edge from `main`
- `:sha-<short>` — exact commit

---

## Configuration

Analyser is configured via a YAML file. A minimal `config.yml`:

```yaml
server:
  # Plugins to load on startup
  plugins:
    - block.so

database:
  driver: postgres        # postgres | mysql | sqlite3 | clickhouse
  host: 127.0.0.1
  port: 5432
  user: postgres
  password: ${DB_PASSWORD}
  name: iotex

iotex:
  chainEndPoint: api.iotex.one:80   # mainnet (use api.testnet.iotex.one:80 for testnet)

blockDB:
  dbPath: chain.db

log:
  zap:
    level: info
```

> **Note:** Never commit real credentials. Use environment variable substitution (`${VAR}`) or a separate untracked config file.

### Bootstrapping with a Snapshot

Indexing from genesis takes time. Speed it up with a pre-indexed `chain.db` snapshot:

- **Mainnet:** https://t.iotex.me/mainnet-data-with-idx-latest
- **Testnet:** https://t.iotex.me/testnet-data-with-idx-latest

Drop the file at the path referenced by `blockDB.dbPath` and Analyser will resume from the snapshot.

---

## Building from Source

### Requirements

- Go 1.23+
- `make`
- A working C linker (plugins use `-buildmode=plugin`, which requires glibc — Alpine without `gcompat` won't work for local builds)

### Build

Clone and build the binary plus all plugins:

```sh
git clone https://github.com/iotexproject/iotex-analyser.git
cd iotex-analyser
make            # binary + every plugin under plugins/
```

Or build individual targets:

```sh
make build              # binary only
make plugin name=block  # one plugin (produces block.so)
make dev                # binary + all plugins with -race for development
```

---

## Running the Server

Start the server:

```sh
./iotex-analyser -c config.yml server
```

Manage plugins at runtime via a second process (the running server picks up changes automatically):

```sh
./iotex-analyser -c config.yml plugin load   simple.so
./iotex-analyser -c config.yml plugin unload simple.so
./iotex-analyser -c config.yml plugin info
```

---

## Plugin System

Each plugin is a Go package compiled to a `.so` file with `-buildmode=plugin`, exposing a single `Plugin` variable that satisfies the `Adapter` interface:

```go
type Adapter interface {
    Name() string
    Version() string
    Type() Type
    Start(context.Context) error
    Stop(context.Context) error
    PutBlock(context.Context, *block.Block) error
}
```

For better throughput, plugins may also implement `PutBlocks(ctx, []*block.Block) error` to process blocks in batches.

### Writing a Plugin

```go
package main

import (
    "context"
    "fmt"

    "github.com/iotexproject/iotex-analyser/db"
    "github.com/iotexproject/iotex-analyser/plugin"
    "github.com/iotexproject/iotex-core/v2/blockchain/block"
)

type simplePlugin struct{}

func (simplePlugin) Name() string         { return "simple" }
func (simplePlugin) Version() string      { return "0.0.1" }
func (simplePlugin) Type() plugin.Type    { return plugin.TypeStandard }
func (simplePlugin) Start(context.Context) error { return nil }
func (simplePlugin) Stop(context.Context) error  { return nil }

func (s simplePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
    fmt.Printf("block height: %d\n", blk.Height())
    return db.UpdateIndexHeight(s.Name(), blk.Height())
}

// Exported symbol — must be named "Plugin".
var Plugin = simplePlugin{}
```

Build it:

```sh
make plugin name=simple
```

### Loading and Unloading at Runtime

```sh
./iotex-analyser -c config.yml plugin load   simple.so
./iotex-analyser -c config.yml plugin unload simple.so
```

After `load`, you should see output from your plugin in the server logs, e.g.:

```
block height: 1
block height: 2
block height: 3
```

### Built-in Plugins

A non-exhaustive sample — see [`plugins/`](plugins/) for the full list:

- **Core blocks/actions:** [block](plugins/block/), [block_action](plugins/block_action/), [block_meta](plugins/block_meta/), [block_supply](plugins/block_supply/), [block_receipts](plugins/block_receipts/)
- **Tokens:** [erc20](plugins/erc20/), [erc721](plugins/erc721/), [erc1155](plugins/erc1155/), token holder/meta variants
- **Staking & delegation:** [candidate](plugins/candidate/), [staking_action](plugins/staking_action/), [delegate](plugins/delegate/), [probation](plugins/probation/)
- **Accounting:** [account_meta](plugins/account_meta/), [account_income](plugins/account_income/), [account_vote](plugins/account_vote/)
- **Other:** [gastracker](plugins/gastracker/), [erc1967_proxy](plugins/erc1967_proxy/), [clickhouse](plugins/clickhouse/) sinks

---

## GraphQL API

[Hasura](https://hasura.io/) sits in front of the indexed Postgres database and serves a GraphQL API automatically.

- **Hasura Console:** https://iotexscout.io/hasura/console
- **GraphQL endpoint:** https://iotexscout.io/hasura/v1/graphql

---

## Releases

Releases are cut manually using [`release.sh`](release.sh):

```sh
./release.sh v1.18.1
```

The script validates the working tree, tags the commit, pushes the tag, and creates a GitHub Release with auto-generated notes. The tag push triggers GitHub Actions to build and publish the Docker image as both `:vX.Y.Z` and `:latest` on `ghcr.io`.

---

## Contributing

Contributions are welcome. The general flow:

1. Open an issue to discuss non-trivial changes before starting work.
2. Fork and create a feature branch.
3. Add tests for new functionality where possible.
4. Open a PR against `main`.

Useful commands while developing:

```sh
make dev          # build with -race for local testing
go test ./...     # run unit tests
```

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).
