package service

import (
	"billing3/database"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"log/slog"
	"regexp"
)

type OrderRequest struct {
	ProductID int               `json:"product_id" validate:"required"`
	Duration  int               `json:"duration" validate:"min=0"`
	Options   map[string]string `json:"options"`
}

type Pricing struct {
	Duration     int             `json:"duration"`
	BillingCycle string          `json:"billing_cycle"`
	RecurringFee decimal.Decimal `json:"recurring_fee"`
	SetupFee     decimal.Decimal `json:"setup_fee"`
	Items        []PricingItem   `json:"items"`
}

type PricingItem struct {
	Price       decimal.Decimal `json:"price"`
	Description string          `json:"description"`
}

// CalculatePricing calculates price for given billing cycle, and configurable options.
// CalculatePricing returns error if product is disabled or out of stock.
// CalculatePricing returns (product, cleaned options, redacted options(with password
// removed, used for logging), pricing, error)
func CalculatePricing(ctx context.Context, req OrderRequest) (*database.Product, map[string]string, map[string]string, *Pricing, error) {
	product, err := database.Q.FindProductById(ctx, int32(req.ProductID))
	if err != nil {
		slog.Error("find product", "err", err, "id", req.ProductID)
		return nil, nil, nil, nil, ErrInternalError
	}

	// product must be enabled
	if !product.Enabled {
		return nil, nil, nil, nil, fmt.Errorf("product is disabled")
	}

	// product must be in stock
	if product.StockControl == StockControlEnabled && product.Stock <= 0 {
		return nil, nil, nil, nil, fmt.Errorf("product is out of stock")
	}

	pricing := Pricing{
		RecurringFee: decimal.NewFromInt(0),
		SetupFee:     decimal.NewFromInt(0),
		Items:        make([]PricingItem, 0),
		Duration:     req.Duration,
	}

	// product pricing

	found := false
	for _, price := range product.Pricing {
		if price.Duration == int32(req.Duration) {
			found = true

			pricing.RecurringFee = pricing.RecurringFee.Add(price.Price)
			pricing.Items = append(pricing.Items, PricingItem{
				Description: product.Name,
				Price:       price.Price,
			})

			if price.SetupFee.GreaterThan(decimal.Zero) {
				pricing.SetupFee = pricing.SetupFee.Add(price.SetupFee)
				pricing.Items = append(pricing.Items, PricingItem{
					Description: product.Name + " Setup Fee",
					Price:       price.SetupFee,
				})
			}

			pricing.BillingCycle = price.DisplayName

			break
		}
	}
	if !found {
		return nil, nil, nil, nil, fmt.Errorf("invalid billing cycle")
	}

	options, err := database.Q.FindProductOptionsByProduct(ctx, int32(req.ProductID))
	if err != nil {
		slog.Error("find options", "err", err, "id", req.ProductID)
		return nil, nil, nil, nil, ErrInternalError
	}

	// validate options

	cleanedOptions := make(map[string]string)
	redactedOptions := make(map[string]string)

	for _, option := range options {
		userInput, ok := req.Options[option.Name]
		if !ok {
			userInput = ""
		}

		if option.Type == "select" {
			// user input must be a valid selection

			found := false
			for _, optionValue := range option.Values {
				for _, price := range optionValue.Prices {
					if price.Duration == int32(req.Duration) {
						found = true
						break
					}
				}
			}
			if !found {
				// this option is unavailable because none of its values has the same duration as req.Duration
				// so we ignore this option
				break
			}

			found = false
			for _, optionValue := range option.Values {
				if optionValue.Value != userInput {
					continue
				}
				found = true

				// find pricing for selected billing cycle
				pricingFound := false
				for _, price := range optionValue.Prices {
					if price.Duration != int32(req.Duration) {
						continue
					}

					if price.Price.GreaterThan(decimal.Zero) {
						pricing.Items = append(pricing.Items, PricingItem{
							Description: "\u00BB " + option.DisplayName + ": " + optionValue.DisplayName,
							Price:       price.Price,
						})
						pricing.RecurringFee = pricing.RecurringFee.Add(price.Price)
					}

					if price.SetupFee.GreaterThan(decimal.Zero) {
						pricing.Items = append(pricing.Items, PricingItem{
							Description: "\u00BB " + option.DisplayName + ": " + optionValue.DisplayName + " Setup Fee",
							Price:       price.SetupFee,
						})
						pricing.SetupFee = pricing.SetupFee.Add(price.SetupFee)
					}

					pricingFound = true
					break
				}

				if !pricingFound {
					return nil, nil, nil, nil, fmt.Errorf("option \"%s\" is not available for the selected billing cycle", option.DisplayName)
				}

				break
			}

			if !found {
				return nil, nil, nil, nil, fmt.Errorf("option \"%s\" has an invalid selection", option.DisplayName)
			}

		} else {

			// validate regex

			if option.Regex != "" {
				compiled, err := regexp.Compile(option.Regex)
				if err != nil {
					slog.Error("invalid regex", "err", err, "regex", option.Regex, "product", req.ProductID, "option", option.Name)
					return nil, nil, nil, nil, ErrInternalError
				}
				if !compiled.MatchString(userInput) {
					return nil, nil, nil, nil, fmt.Errorf("option \"%s\" is invalid", option.DisplayName)
				}
			}

		}

		if option.Type == "password" {
			cleanedOptions[option.Name] = userInput
			redactedOptions[option.Name] = "******"
		} else {
			cleanedOptions[option.Name] = userInput
			redactedOptions[option.Name] = userInput
		}
	}

	return &product, cleanedOptions, redactedOptions, &pricing, nil
}
