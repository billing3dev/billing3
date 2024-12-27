package controller

import (
	"billing3/database"
	"billing3/database/types"
	"billing3/service"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"strconv"
)

func listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := database.Q.ListCategories(r.Context())
	if err != nil {
		slog.Error("list category", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"categories": categories})
}

func getCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	category, err := database.Q.FindCategoryById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("get category", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"category": category})
}

func listProductByCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	products, err := database.Q.FindEnabledProductsByCategory(r.Context(), int32(id))
	if err != nil {
		slog.Error("list products", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type productStruct struct {
		ID          int32               `json:"id"`
		Name        string              `json:"name"`
		Description string              `json:"description"`
		Pricing     types.ProductPrices `json:"pricing"`
		InStock     bool                `json:"in_stock"`
	}

	resp := make([]productStruct, 0)
	for _, p := range products {
		resp = append(resp, productStruct{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Pricing:     p.Pricing,
			InStock:     p.StockControl == service.StockControlDisabled || p.Stock > 0,
		})
	}

	writeResp(w, http.StatusOK, D{"products": resp})
}

func getProduct(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("get product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !product.Enabled {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type productStruct struct {
		ID          int32               `json:"id"`
		Name        string              `json:"name"`
		Description string              `json:"description"`
		Pricing     types.ProductPrices `json:"pricing"`
		InStock     bool                `json:"in_stock"`
	}
	p := productStruct{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Pricing:     product.Pricing,
		InStock:     product.StockControl == service.StockControlDisabled || product.Stock > 0,
	}

	writeResp(w, http.StatusOK, D{"product": p})
}

func getProductOptions(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// billing cycle
	duration, err := strconv.Atoi(r.URL.Query().Get("duration"))
	if err != nil || duration < 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check if product is enabled
	product, err := database.Q.FindProductById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("get product", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !product.Enabled {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// check if product is in stock
	if product.StockControl == service.StockControlEnabled && product.Stock <= 0 {
		writeError(w, http.StatusBadRequest, "product is out of stock")
		return
	}

	allOptions, err := database.Q.FindProductOptionsByProduct(r.Context(), int32(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("get product options", "err", err)
		return
	}

	// filter options based on billing cycle
	filteredOptions := make([]database.ProductOption, 0)

	for _, option := range allOptions {

		if option.Type != "select" {
			filteredOptions = append(filteredOptions, option)
			continue
		}

		ok := false // whether this select has at least one valid value

		filteredOption := database.ProductOption{
			ProductID:   option.ProductID,
			Name:        option.Name,
			DisplayName: option.DisplayName,
			Type:        option.Type,
			Regex:       option.Regex,
			Description: option.Description,
			Values:      make(types.ProductOptionValues, 0),
		}

		for _, value := range option.Values {

			for _, price := range value.Prices {

				// a value is considered valid if one of its pricings has the same billing cycle as the product
				if price.Duration == int32(duration) {

					ok = true

					filteredOption.Values = append(filteredOption.Values, types.ProductOptionValue{
						DisplayName: value.DisplayName,
						Value:       value.Value,
						Prices: []types.ProductOptionValuePrice{
							{Duration: price.Duration, Price: price.Price, SetupFee: price.SetupFee},
						},
					})

					break
				}
			}

		}

		// a selection must at least one valid value
		if ok {
			filteredOptions = append(filteredOptions, filteredOption)
		}
	}

	writeResp(w, http.StatusOK, D{"options": filteredOptions})
}
