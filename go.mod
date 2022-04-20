module github.com/iotexproject/iotex-analyser

go 1.16

require (
	github.com/ethereum/go-ethereum v1.10.17
	github.com/imdario/mergo v0.3.12
	github.com/iotexproject/go-pkgs v0.1.12-0.20220209063039-b876814568a0
	github.com/iotexproject/iotex-address v0.2.6
	github.com/iotexproject/iotex-core v1.7.1
	github.com/iotexproject/iotex-election v0.3.5-0.20210611041425-20ddf674363d
	github.com/iotexproject/iotex-proto v0.5.9
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/millken/gocache v1.0.5
	github.com/mitchellh/go-homedir v1.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rodaine/table v1.0.1
	github.com/sethvargo/go-envconfig v0.5.0
	github.com/shopspring/decimal v1.3.1
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/mysql v1.2.3
	gorm.io/driver/postgres v1.2.3
	gorm.io/driver/sqlite v1.2.6
	gorm.io/gorm v1.22.5
)

// replace github.com/ethereum/go-ethereum => github.com/iotexproject/go-ethereum v0.3.1
