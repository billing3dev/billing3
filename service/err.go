package service

import "errors"

var ErrServiceCancelled = errors.New("service is cancelled")
var ErrUnpaidInvoiceExists = errors.New("unpaid invoice already exists for the service")
var ErrNotFound = errors.New("not found")
var ErrInternalError = errors.New("internal error")
