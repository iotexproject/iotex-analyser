# iotex-analyser
async analyser for iotex

## Install

```
make plugin #auto make plugin so file
make build #server
make run #build + run
make #both plugin and plugin
```

## Usage

```
./iotex-analyser -c config.yml server #start server
./iotex-analyser -c config.yml plugin load block.so #load plugin
./iotex-analyser -c config.yml plugin unload block.so #unload plugin
```

### simple config.yml
```
database:
  dsn: root:root@tcp(127.0.0.1:3306)/iotex-analyser
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