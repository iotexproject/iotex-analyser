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
```