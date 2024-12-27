package controller

import (
	"billing3/database"
	"billing3/database/types"
	"billing3/service/extension"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type adminProductReqStruct struct {
	Name         string              `json:"name" validate:"required"`
	Description  string              `json:"description"`
	CategoryId   int32               `json:"category_id" validate:"required"`
	Extension    string              `json:"extension" validate:"required"`
	Enabled      bool                `json:"enabled"`
	Pricing      types.ProductPrices `json:"pricing"`
	Settings     map[string]string   `json:"settings"`
	Stock        int32               `json:"stock" validate:"min=0"`
	StockControl int32               `json:"stock_control" validate:"oneof=1 2"`
	Options      []struct {
		DisplayName string                    `json:"display_name" validate:"required"`
		Name        string                    `json:"name" validate:"required"`
		Description string                    `json:"description"`
		Regex       string                    `json:"regex"`
		Type        string                    `json:"type" validate:"required,oneof=select textarea password text"`
		Values      types.ProductOptionValues `json:"values"`
	} `json:"options" validate:"dive"`
}

func adminProductList(w http.ResponseWriter, r *http.Request) {
	categoryId, err := strconv.Atoi(r.URL.Query().Get("category"))
	if err != nil {
		categoryId = 0
	}

	type productStruct struct {
		ID           int32                 `json:"id"`
		Name         string                `json:"name"`
		Description  string                `json:"description"`
		CategoryID   int32                 `json:"category_id"`
		CategoryName string                `json:"category_name"`
		Extension    string                `json:"extension"`
		Enabled      bool                  `json:"enabled"`
		Pricing      types.ProductPrices   `json:"pricing"`
		Settings     types.ProductSettings `json:"settings"`
		Stock        int32                 `json:"stock"`
		StockControl int32                 `json:"stock_control"`
	}

	products := make([]productStruct, 0)

	products2, err := database.Q.SearchProduct(r.Context(), int32(categoryId))
	if err != nil {
		slog.Error("admin search products", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, p := range products2 {
		products = append(products, productStruct{
			ID:           p.ID,
			Name:         p.Name,
			Description:  p.Description,
			CategoryID:   p.CategoryID,
			CategoryName: p.CategoryName,
			Extension:    p.Extension,
			Enabled:      p.Enabled,
			Pricing:      p.Pricing,
			Settings:     p.Settings,
			Stock:        p.Stock,
			StockControl: p.StockControl,
		})
	}

	writeResp(w, http.StatusOK, D{"products": products})
}

// returns cleaned extension settings
func adminProductReqValidate(req *adminProductReqStruct) (map[string]string, error) {
	// validation
	ext, ok := extension.Extensions[req.Extension]
	if !ok {
		return nil, fmt.Errorf("unknown extension: %s", req.Extension)
	}

	if len(req.Pricing) == 0 {
		return nil, fmt.Errorf("at least one pricing is required")
	}

	pricingDisplayNames := make(map[string]bool) // set of pricing display names
	pricingDurations := make(map[int32]bool)     // set of pricing durations
	for _, p := range req.Pricing {
		if p.DisplayName == "" {
			return nil, fmt.Errorf("pricing display name is required")
		}

		if !p.Price.GreaterThanOrEqual(decimal.NewFromInt(0)) {
			return nil, fmt.Errorf("price must not be negative")
		}

		if !p.SetupFee.GreaterThanOrEqual(decimal.NewFromInt(0)) {
			return nil, fmt.Errorf("setup fee must not be negative")
		}

		// pricing display names must be unique
		if _, ok := pricingDisplayNames[p.DisplayName]; ok {
			return nil, fmt.Errorf("duplicated pricing: %s", p.DisplayName)
		}

		// pricing duration must be unique
		if _, ok := pricingDurations[p.Duration]; ok {
			return nil, fmt.Errorf("duplicated duration: %s", p.DisplayName)
		}

		pricingDurations[p.Duration] = true
		pricingDisplayNames[p.DisplayName] = true
	}

	// validating product settings
	settings, err := ext.ProductSettings(req.Settings)
	if err != nil {
		slog.Error("create product: ext.ProductSettings", "err", err, "extension", req.Extension, "inputs", req.Settings)
		return nil, fmt.Errorf("internal error: could not get setting list")
	}

	// validated settings against rules returned by ext.ProductSettings
	cleanedSettings := make(map[string]string)

	for _, setting := range settings {
		input, ok := req.Settings[setting.Name]
		if !ok {
			input = ""
		}

		if setting.Type == "select" {
			// input for selection must be present in setting.Values
			if !slices.Contains(setting.Values, input) {
				return nil, fmt.Errorf("invalid extension setting: %s", setting.DisplayName)
			}
		} else if setting.Type == "servers" {
			for _, s := range strings.Split(input, ",") {
				i, err := strconv.Atoi(s)
				if err != nil || i < 0 {
					return nil, fmt.Errorf("invalid servers")
				}
			}
		} else {
			// skip if regex is not provided
			if setting.Regex != "" {
				compiledRegex, err := regexp.Compile(setting.Regex)
				if err != nil {
					slog.Error("invalid regex", "err", err, "regex", setting.Regex, "extension", req.Extension, "setting", setting.Name)
					return nil, fmt.Errorf("internal error: invalid regex")
				}

				if !compiledRegex.Match([]byte(input)) {
					return nil, fmt.Errorf("invalid extension setting: %s", setting.DisplayName)
				}
			}
		}

		cleanedSettings[setting.Name] = input
	}

	// validate options

	for _, option := range req.Options {
		if option.Type != "select" {

			// non-select must not have any values
			option.Values = []types.ProductOptionValue{}

		} else {

			// validate values for selection
			for _, value := range option.Values {

				if value.DisplayName == "" {
					return nil, fmt.Errorf("option \"%s\" has a value with missing display name", option.DisplayName)
				}

				if len(value.Prices) == 0 {
					return nil, fmt.Errorf("option \"%s\" has a value with missing prices", option.DisplayName)
				}

				for _, price := range value.Prices {
					if price.SetupFee.LessThan(decimal.NewFromInt(0)) || price.Price.LessThan(decimal.NewFromInt(0)) {
						return nil, fmt.Errorf("price and setup fee must not be negative")
					}

					found := false
					for _, productPrice := range req.Pricing {
						if productPrice.Duration == price.Duration {
							found = true
							break
						}
					}
					if !found {
						return nil, fmt.Errorf("billing cycle of option must be one of product's billing cycles")
					}
				}

			}
		}
	}

	return cleanedSettings, nil
}

func adminProductCreate(w http.ResponseWriter, r *http.Request) {
	req, err := decode[adminProductReqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedProductSettings, err := adminProductReqValidate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// insert into database

	tx, err := database.Conn.Begin(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("create product: begin tx", "err", err)
		return
	}
	defer rollbackTx(r.Context(), tx)
	qtx := database.Q.WithTx(tx)

	// insert product
	id, err := qtx.CreateProduct(r.Context(), database.CreateProductParams{
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		Extension:    req.Extension,
		Enabled:      req.Enabled,
		Pricing:      req.Pricing,
		Settings:     cleanedProductSettings,
		Stock:        req.Stock,
		StockControl: req.StockControl,
	})
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == PGErrorUniqueViolation {
			writeError(w, http.StatusForbidden, "duplicated product name")
			return
		}
		slog.Error("admin create product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// insert options
	for _, option := range req.Options {
		err = qtx.CreateProductOption(r.Context(), database.CreateProductOptionParams{
			ProductID:   id,
			Name:        option.Name,
			Description: option.Description,
			DisplayName: option.DisplayName,
			Type:        option.Type,
			Regex:       option.Regex,
			Values:      option.Values,
		})
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == PGErrorUniqueViolation {
				writeError(w, http.StatusBadRequest, "duplicated option name: "+option.Name)
				return
			}
			slog.Error("admin create product option", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin create product: commit tx", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{"id": id})
}

func adminProductUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req, err := decode[adminProductReqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedProductSettings, err := adminProductReqValidate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// update database
	tx, err := database.Conn.Begin(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("create product: begin tx", "err", err)
		return
	}
	defer rollbackTx(r.Context(), tx)
	qtx := database.Q.WithTx(tx)

	// update product
	err = qtx.UpdateProduct(r.Context(), database.UpdateProductParams{
		ID:           int32(id),
		Name:         req.Name,
		Description:  req.Description,
		CategoryID:   req.CategoryId,
		Extension:    req.Extension,
		Enabled:      req.Enabled,
		Pricing:      req.Pricing,
		Settings:     cleanedProductSettings,
		Stock:        req.Stock,
		StockControl: req.StockControl,
	})
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == PGErrorUniqueViolation {
			writeError(w, http.StatusForbidden, "Duplicated product name")
			return
		}
		slog.Error("admin update product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// delete old product options
	err = qtx.DeleteProductOptionsByProduct(r.Context(), int32(id))
	if err != nil {
		slog.Error("admin update product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// insert product options
	for _, option := range req.Options {
		err = qtx.CreateProductOption(r.Context(), database.CreateProductOptionParams{
			ProductID:   int32(id),
			Name:        option.Name,
			Description: option.Description,
			DisplayName: option.DisplayName,
			Type:        option.Type,
			Regex:       option.Regex,
			Values:      option.Values,
		})
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == PGErrorUniqueViolation {
				writeError(w, http.StatusBadRequest, "duplicated option name: "+option.Name)
				return
			}
			slog.Error("admin update product option", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// commit tx
	err = tx.Commit(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin update product: commit tx", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminProductDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = database.Q.DeleteProductOptionsByProduct(r.Context(), int32(id))
	if err != nil {
		slog.Error("admin delete product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = database.Q.DeleteProduct(r.Context(), int32(id))
	if err != nil {
		slog.Error("admin delete product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminProductGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	product, err := database.Q.FindProductById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("admin get product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	options, err := database.Q.FindProductOptionsByProduct(r.Context(), int32(product.ID))
	if err != nil {
		slog.Error("admin get product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"product": map[string]any{
		"id":            product.ID,
		"name":          product.Name,
		"description":   product.Description,
		"category_id":   product.CategoryID,
		"extension":     product.Extension,
		"enabled":       product.Enabled,
		"pricing":       product.Pricing,
		"settings":      product.Settings,
		"options":       options,
		"stock":         product.Stock,
		"stock_control": product.StockControl,
	}})
}

// returns extension settings based on user inputs
func adminProductExtensionSettings(w http.ResponseWriter, r *http.Request) {
	type reqStruct struct {
		Extension string `json:"extension" validate:"required"`
		Inputs    map[string]string
	}

	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	e, ok := extension.Extensions[req.Extension]
	if !ok {
		writeError(w, http.StatusBadRequest, "Extension not found")
		return
	}

	settings, err := e.ProductSettings(req.Inputs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "extension: "+err.Error())
		return
	}

	writeResp(w, http.StatusOK, D{
		"settings": settings,
	})
}

func adminProductExtensionList(w http.ResponseWriter, r *http.Request) {
	list := make([]string, 0)

	for k, _ := range extension.Extensions {
		list = append(list, k)
	}

	writeResp(w, http.StatusOK, D{"extensions": list})
}
