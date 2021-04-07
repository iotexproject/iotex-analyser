package main

import (
	"database/sql/driver"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/iotexproject/iotex-analyser/kernel"
)

const (
	CryptoLatestPriceURL   = "https://crypto.com/price/coin-data/iotex/basic_info.json"
	CryptoLatestPrice1dURL = "https://crypto.com/price/coin-data/iotex/1d/latest.json"
)

type cryptoLatestPrice struct {
	BtcMarketcap      float64 `json:"btc_marketcap"`
	BtcPriceChange    float64 `json:"btc_price_change"`
	BtcPriceCurrent   float64 `json:"btc_price_current"`
	BtcVolume24H      float64 `json:"btc_volume_24h"`
	CirculatingSupply float64 `json:"circulating_supply"`
	ConverterInfo     struct {
		BtcUsdPriceCurrent float64 `json:"btc_usd_price_current"`
		CroUsdPriceCurrent float64 `json:"cro_usd_price_current"`
		EthUsdPriceCurrent float64 `json:"eth_usd_price_current"`
		McoUsdPriceCurrent float64 `json:"mco_usd_price_current"`
	} `json:"converter_info"`
	MaxSupply       json.Number `json:"max_supply"`
	Rank            int         `json:"rank"`
	UsdMarketcap    float64     `json:"usd_marketcap"`
	UsdPriceChange  float64     `json:"usd_price_change"`
	UsdPriceCurrent float64     `json:"usd_price_current"`
	UsdVolume24H    float64     `json:"usd_volume_24h"`
}

func (m *cryptoLatestPrice) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return driver.Value([]byte(j)), nil
}

type latestPrice1d struct {
	Data        [][]float64 `json:"data"`
	PriceChange float64     `json:"price_change"`
	Timestamp   time.Time   `json:"timestamp"`
}

func (m *latestPrice1d) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return driver.Value([]byte(j)), nil
}

func priceFetcher() (*cryptoLatestPrice, error) {
	var info cryptoLatestPrice
	response, err := kernel.DefaultHTTPClient.Get(CryptoLatestPriceURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func price1dFetcher() (*latestPrice1d, error) {
	var info latestPrice1d
	response, err := kernel.DefaultHTTPClient.Get(CryptoLatestPrice1dURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
