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

type euroHandler struct {
	logger          log.Logger
	currencyService currency.Service
}

func NewEuroHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*euroHandler, error) {
	currencyService, err := currency.New(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create currencyService: %w", err)
	}

	return &euroHandler{
		logger:          logger,
		currencyService: currencyService,
	}, nil
}

func (h *euroHandler) Handle(requestContext context.Context, request *httpserver.Request) (response *httpserver.Response, err error) {
	currency := request.Params.ByName("currency")
	amountString := request.Params.ByName("amount")
	amount, err := strconv.ParseFloat(amountString, 64)
	if err != nil {
		h.logger.Error("cannot parse amount %s: %w", amountString, err)

		return httpserver.NewStatusResponse(http.StatusBadRequest), nil
	}

	result, err := h.currencyService.ToEur(requestContext, amount, currency)
	if err != nil {
		h.logger.Error("cannot convert amount %f with currency %s: %w", amount, currency, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	return httpserver.NewJsonResponse(result), nil
}

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

func (h *euroAtDateHandler) Handle(requestContext context.Context, request *httpserver.Request) (response *httpserver.Response, err error) {
	currency := request.Params.ByName("currency")
	dateString := request.Params.ByName("date")
	date, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		h.logger.Error("cannot parse date %s: %w", dateString, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	amountString := request.Params.ByName("amount")
	amount, err := strconv.ParseFloat(amountString, 64)
	if err != nil {
		h.logger.Error("cannot parse amount %s: %w", amountString, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	result, err := h.currencyService.ToEurAtDate(requestContext, amount, currency, date)
	if err != nil {
		h.logger.Error("cannot convert amount %f with currency %s at date %v: %w", amount, currency, date, err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	return httpserver.NewJsonResponse(result), nil
}
