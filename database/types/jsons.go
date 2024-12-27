package types

import "github.com/shopspring/decimal"

type ProductPrices = []ProductPrice

type ProductPrice struct {
	DisplayName string          `json:"display_name"`
	Duration    int32           `json:"duration"` // number of seconds
	Price       decimal.Decimal `json:"price"`
	SetupFee    decimal.Decimal `json:"setup_fee"`
}

type ProductOptionValues = []ProductOptionValue

type ProductOptionValuePrice struct {
	Duration int32           `json:"duration"` // number of seconds
	Price    decimal.Decimal `json:"price"`
	SetupFee decimal.Decimal `json:"setup_fee"`
}

type ProductOptionValue struct {
	Value       string                    `json:"value"`
	DisplayName string                    `json:"display_name"`
	Prices      []ProductOptionValuePrice `json:"prices"`
}

type ProductSettings map[string]string

type ServiceSettings map[string]string

type GatewaySettings map[string]string

type ServerSettings map[string]string
