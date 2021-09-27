module github.com/iotexproject/iotex-analyser

go 1.14

require (
	github.com/dustinxie/gmsm v1.2.1 // indirect
	github.com/ethereum/go-ethereum v1.10.8
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/imdario/mergo v0.3.12
	github.com/iotexproject/go-pkgs v0.1.5
	github.com/iotexproject/iotex-address v0.2.5
	github.com/iotexproject/iotex-core v1.3.0-rc1.0.20210907215229-0b65bf08ac88
	github.com/iotexproject/iotex-election v0.3.5-0.20210611041425-20ddf674363d
	github.com/iotexproject/iotex-proto v0.5.2
	github.com/lib/pq v1.10.2 // indirect
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rodaine/table v1.0.1
	github.com/sethvargo/go-envconfig v0.3.5
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	go.uber.org/zap v1.16.0
	google.golang.org/genproto v0.0.0-20210224155714-063164c882e6 // indirect
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0-rc.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	gorm.io/driver/mysql v1.1.0
	gorm.io/driver/postgres v1.1.0
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.10
)

// replace github.com/ethereum/go-ethereum => github.com/iotexproject/go-ethereum v0.3.1
