package main

import (
	"database/sql/driver"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/iotexproject/iotex-analyser/kernel"
)

const (
	PriceURL = "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=iotex&order=market_cap_desc&per_page=100&page=1&sparkline=false"
)

type priceCoin struct {
	ID                           string      `json:"id"`
	Symbol                       string      `json:"symbol"`
	Name                         string      `json:"name"`
	Image                        string      `json:"image"`
	CurrentPrice                 float64     `json:"current_price"`
	MarketCap                    int         `json:"market_cap"`
	MarketCapRank                int         `json:"market_cap_rank"`
	FullyDilutedValuation        int         `json:"fully_diluted_valuation"`
	TotalVolume                  int         `json:"total_volume"`
	High24H                      float64     `json:"high_24h"`
	Low24H                       float64     `json:"low_24h"`
	PriceChange24H               float64     `json:"price_change_24h"`
	PriceChangePercentage24H     float64     `json:"price_change_percentage_24h"`
	MarketCapChange24H           int         `json:"market_cap_change_24h"`
	MarketCapChangePercentage24H float64     `json:"market_cap_change_percentage_24h"`
	CirculatingSupply            float64     `json:"circulating_supply"`
	TotalSupply                  float64     `json:"total_supply"`
	MaxSupply                    float64     `json:"max_supply"`
	Ath                          float64     `json:"ath"`
	AthChangePercentage          float64     `json:"ath_change_percentage"`
	AthDate                      time.Time   `json:"ath_date"`
	Atl                          float64     `json:"atl"`
	AtlChangePercentage          float64     `json:"atl_change_percentage"`
	AtlDate                      time.Time   `json:"atl_date"`
	Roi                          interface{} `json:"roi"`
	LastUpdated                  time.Time   `json:"last_updated"`
}

type priceCoins []*priceCoin

func (m *priceCoin) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return driver.Value([]byte(j)), nil
}

func priceFetcher() (*priceCoin, error) {
	var info priceCoins
	var err error
	response, err := kernel.DefaultHTTPClient.Get(PriceURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	// body := []byte(`[{"id":"iotex","symbol":"iotx","name":"IoTeX","image":"https://assets.coingecko.com/coins/images/3334/large/iotex-logo.png?1547037941","current_price":0.064283,"market_cap":611449352,"market_cap_rank":156,"fully_diluted_valuation":611649827,"total_volume":75758611,"high_24h":0.06768,"low_24h":0.060147,"price_change_24h":0.00274793,"price_change_percentage_24h":4.46563,"market_cap_change_24h":40312812,"market_cap_change_percentage_24h":7.05835,"circulating_supply":9493154322.48387,"total_supply":9496266827.32,"max_supply":9496266827.32,"ath":0.14167,"ath_change_percentage":-54.53546,"ath_date":"2021-08-12T01:40:07.028Z","atl":0.00121576,"atl_change_percentage":5197.88814,"atl_date":"2020-03-13T02:29:47.597Z","roi":null,"last_updated":"2021-10-29T02:28:19.654Z"}]`)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return info[0], nil
}
