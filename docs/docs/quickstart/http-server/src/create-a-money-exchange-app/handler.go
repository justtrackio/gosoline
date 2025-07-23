// snippet-start: imports
package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/currency"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

// snippet-end: imports

// snippet-start: euro handler
type euroHandler struct {
	logger          log.Logger
	currencyService currency.Service
}

// snippet-end: euro handler

// snippet-start: new euro handler
func NewEuroHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*euroHandler, error) {
	// Instantiate a new currencyService
	currencyService, err := currency.New(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create currencyService: %w", err)
	}

	// Return a euroHandler
	return &euroHandler{
		logger:          logger,
		currencyService: currencyService,
	}, nil
}

// snippet-end: new euro handler

// snippet-start: handle
func (h *euroHandler) Handle(requestContext context.Context, request *httpserver.Request) (response *httpserver.Response, err error) {
	// Get a currency and amountString from the request parameters.
	currency := request.Params.ByName("currency")
	amountString := request.Params.ByName("amount")

	// Parse a float value from amountString.
	amount, err := strconv.ParseFloat(amountString, 64)
	// Send a 400 Bad Request response if amountString can't be parsed into a valid float.
	if err != nil {
		h.logger.Error(requestContext, "cannot parse amount %s: %w", amountString, err)

		return httpserver.NewStatusResponse(http.StatusBadRequest), nil
	}

	// Convert the amount from the source currency to euros.
	result, err := h.currencyService.ToEur(requestContext, amount, currency)
	// Send a 500 Internal Server Error if the server can't convert the amount.
	if err != nil {
		h.logger.Error(requestContext, "cannot convert amount %f with currency %s: %w", amount, currency, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	// Send a 200 OK Json response back to the client with the results.
	return httpserver.NewJsonResponse(result), nil
}

// snippet-end: handle

// snippet-start: euro at date handler
type euroAtDateHandler struct {
	logger          log.Logger
	currencyService currency.Service
}

func NewEuroAtDateHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*euroAtDateHandler, error) {
	currencyService, err := currency.New(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create currencyService: %w", err)
	}

	return &euroAtDateHandler{
		logger:          logger,
		currencyService: currencyService,
	}, nil
}

// snippet-end: euro at date handler

// snippet-start: euro-at-date handler handle
func (h *euroAtDateHandler) Handle(requestContext context.Context, request *httpserver.Request) (response *httpserver.Response, err error) {
	// Get the request parameters and parse their string values.
	currency := request.Params.ByName("currency")
	dateString := request.Params.ByName("date")
	date, err := time.Parse(time.RFC3339, dateString)
	// Send a 500 Internal Server Error if the service can't parse the params or convert the currency.
	if err != nil {
		h.logger.Error(requestContext, "cannot parse date %s: %w", dateString, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	amountString := request.Params.ByName("amount")
	amount, err := strconv.ParseFloat(amountString, 64)
	if err != nil {
		h.logger.Error(requestContext, "cannot parse amount %s: %w", amountString, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	result, err := h.currencyService.ToEurAtDate(requestContext, amount, currency, date)
	if err != nil {
		h.logger.Error(requestContext, "cannot convert amount %f with currency %s at date %v: %w", amount, currency, date, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	// Send a 200 OK Json response back to the client with the results.
	return httpserver.NewJsonResponse(result), nil
}

// snippet-end: euro-at-date handler handle
