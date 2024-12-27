package controller

import (
	"billing3/database"
	"billing3/database/types"
	"billing3/service"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"log/slog"
	"net/http"
	"strconv"
)

func adminInvoiceList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != service.InvoiceUnpaid && status != service.InvoicePaid && status != service.InvoiceCancelled {
		status = ""
	}

	userId, err := strconv.Atoi(r.URL.Query().Get("user"))
	if err != nil {
		userId = 0
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	totalPages, invoices, err := service.SearchInvoice(r.Context(), status, userId, page, itemPerPage)
	if err != nil {
		slog.Error("admin list invoice", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"invoices": invoices, "total_pages": totalPages})
}

func adminInvoiceGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	invoice, err := database.Q.FindInvoiceByIdWithUsername(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("admin get invoice", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	items, err := database.Q.ListInvoiceItems(r.Context(), invoice.ID)
	if err != nil {
		slog.Error("admin get invoice", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"invoice": invoice, "items": items})
}

func adminInvoiceEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		Status             string          `json:"status" validate:"required,oneof=PAID UNPAID CANCELLED"`
		CancellationReason pgtype.Text     `json:"cancellation_reason" `
		PaidAt             types.Timestamp `json:"paid_at"`
		DueAt              types.Timestamp `json:"due_at" validate:"required"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// update invoice
	err = database.Q.UpdateInvoice(r.Context(), database.UpdateInvoiceParams{
		Status:             req.Status,
		CancellationReason: req.CancellationReason,
		PaidAt:             req.PaidAt,
		DueAt:              req.DueAt,
		ID:                 int32(id),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin update invoice", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminInvoiceAddItem(w http.ResponseWriter, r *http.Request) {
	invoiceId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		Description string          `json:"description" validate:"required"`
		Amount      decimal.Decimal `json:"amount"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount.LessThan(decimal.Zero) {
		writeError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}

	err = database.Q.CreateInvoiceItem(r.Context(), database.CreateInvoiceItemParams{
		InvoiceID:   int32(invoiceId),
		Description: req.Description,
		Amount:      req.Amount,
		Type:        service.InvoiceItemNone,
		ItemID:      pgtype.Int4{Valid: false},
	})
	if err != nil {
		slog.Error("admin invoice add item", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = database.Q.UpdateInvoiceAmount(r.Context(), int32(invoiceId))
	if err != nil {
		slog.Error("admin invoice add item", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusCreated, D{})
}

func adminInvoiceRemoveItem(w http.ResponseWriter, r *http.Request) {
	invoiceId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "item_id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = database.Q.DeleteInvoiceItem(r.Context(), database.DeleteInvoiceItemParams{
		ID:        int32(id),
		InvoiceID: int32(invoiceId),
	})
	if err != nil {
		slog.Error("admin invoice remove item", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = database.Q.UpdateInvoiceAmount(r.Context(), int32(invoiceId))
	if err != nil {
		slog.Error("admin invoice remove item", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminInvoiceUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "item_id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	invoiceId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		Description string          `json:"description" validate:"required"`
		Amount      decimal.Decimal `json:"amount"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount.LessThan(decimal.Zero) {
		writeError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}

	err = database.Q.UpdateInvoiceItem(r.Context(), database.UpdateInvoiceItemParams{
		Description: req.Description,
		Amount:      req.Amount,
		ID:          int32(id),
		InvoiceID:   int32(invoiceId),
	})
	if err != nil {
		slog.Error("admin invoice update item", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = database.Q.UpdateInvoiceAmount(r.Context(), int32(invoiceId))
	if err != nil {
		slog.Error("admin invoice remove item", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminListInvoicePayment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	payments, err := database.Q.ListInvoicePayments(r.Context(), int32(id))
	if err != nil {
		slog.Error("admin list invoice payments", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"payments": payments})
}

func adminAddInvoicePayment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		Amount      decimal.Decimal `json:"amount" validate:"required"`
		Description string          `json:"description" validate:"required"`
	}

	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = service.InvoiceAddPayment(r.Context(), int32(id), req.Description, req.Amount, "", "None")
	if err != nil {
		slog.Error("admin add invoice payment", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminInvoiceListByService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	invoices, err := database.Q.FindInvoiceByService(r.Context(), pgtype.Int4{Valid: true, Int32: int32(id)})
	if err != nil {
		slog.Error("admin invoice list by service", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"invoices": invoices})

}
