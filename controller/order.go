package controller

import (
	"billing3/controller/middlewares"
	"billing3/database"
	"billing3/database/types"
	"billing3/service"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"log/slog"
	"net/http"
	"time"
)

func calculatePrice(w http.ResponseWriter, r *http.Request) {
	req, err := decode[service.OrderRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, _, _, pricing, err := service.CalculatePricing(r.Context(), *req)
	if err != nil {
		if errors.Is(err, service.ErrInternalError) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeResp(w, http.StatusOK, D{"pricing": pricing})
}

func order(w http.ResponseWriter, r *http.Request) {
	user := middlewares.MustGetUser(r)

	req, err := decode[service.OrderRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// calculate price
	product, options, redactedOptions, pricing, err := service.CalculatePricing(r.Context(), *req)
	if err != nil {
		if errors.Is(err, service.ErrInternalError) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// start transaction
	tx, err := database.Conn.Begin(r.Context())
	if err != nil {
		slog.Error("begin tx", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rollbackTx(r.Context(), tx)

	qtx := database.Q.WithTx(tx)

	// service settings
	serviceSettings := make(map[string]string)

	for k, v := range product.Settings {
		serviceSettings[k] = v
	}

	// options overwrite product settings
	for k, v := range options {
		serviceSettings[k] = v
	}

	// create service
	serviceId, err := qtx.CreateService(r.Context(), database.CreateServiceParams{
		Label:        product.Name,
		UserID:       user.ID,
		Status:       service.ServiceUnpaid,
		BillingCycle: int32(pricing.Duration),
		Price:        pricing.RecurringFee,
		Extension:    product.Extension,
		Settings:     serviceSettings,
		ExpiresAt:    types.Timestamp{Timestamp: pgtype.Timestamp{Valid: true, Time: time.Now().UTC()}},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("create service", "err", err, "product", req.ProductID)
		return
	}

	// create invoice
	invoiceId, err := service.CreateRenewalInvoice(r.Context(), qtx, serviceId, pricing.SetupFee)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("create invoice", "err", err)
		return
	}
	slog.Debug("invoice created", "id", invoiceId)

	err = tx.Commit(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("commit tx", "err", err)
		return
	}

	slog.Info("new order", "product", product.ID, "label", product.Name, "duration", pricing.Duration, "billing cycle", pricing.BillingCycle, "options", redactedOptions, "product settings", product.Settings, "recurring fee", pricing.RecurringFee, "setup fee", pricing.SetupFee, "user", user.ID, "service id", serviceId, "invoice id", invoiceId)

	writeResp(w, http.StatusOK, D{"invoice": invoiceId})
}
