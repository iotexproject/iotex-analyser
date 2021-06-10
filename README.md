# iotex-analyser
async analyser for iotex

## Install
### protoc

protoc download https://github.com/protocolbuffers/protobuf
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.4
make proto

```

```
make plugins #auto make plugin so file
make plugin name=xxx #compile single plugin,xxx means plugins/xxx/*.go
make build #server
make #both server and plugins
```

## Usage

- please use the snapshot with index data:
   1. mainnet: https://t.iotex.me/mainnet-data-with-idx-latest
   2. testnet: https://t.iotex.me/testnet-data-with-idx-latest

```
./iotex-analyser -c config.yml server #start server
./iotex-analyser -c config.yml plugin load block.so #dynmic load plugin
./iotex-analyser -c config.yml plugin unload block.so #dynmic unload plugin
./iotex-analyser -c config.yml plugin info #display plugin running information
```

### simple config.yml
```
database:
  driver: postgres
  host: 127.0.0.1
  port: 5432
  user: postgres
  password: admin
  name: test
iotex:
  chainEndPoint: api.testnet.iotex.one:80
blockDB:
  dbPath: chain.db
genesis: #testnet is here, mainnet see https://github.com/iotexproject/iotex-bootstrap/blob/master/genesis_mainnet.yaml
  account:
    initBalances:
      io10t7juxazfteqzjsd6qjk7tkgmngj2tm7n4fvrd: "1000000000000000000000000000"
      io120au9ra0nffdle04jx2g5gccn6gq8qd4fy03l4: "7000000000000000000000000000"
      io1yrzvkucxpytn4fru35lc8r8jk4jtue4syg8d4h: "800000000000000000000000000"
```