package crud

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

const HttpStatusClientClosedRequest = 499

var ErrModelNotChanged = fmt.Errorf("nothing has changed on model")

// handleErrorOnWrite handles errors for read operations.
// Covers many default errors and responses like
//   - context.Canceled, context.DeadlineExceed -> HTTP 499
//   - db_repo.RecordNotFoundError | db_repo.NoQueryResultsError -> HTTP 404
//   - validation.Error -> HTTP 400
func handleErrorOnRead(logger log.Logger, err error) (*httpserver.Response, error) {
	if exec.IsRequestCanceled(err) {
		logger.Info("read model(s) aborted: %s", err.Error())

		return httpserver.NewStatusResponse(HttpStatusClientClosedRequest), nil
	}

	if db_repo.IsRecordNotFoundError(err) || db_repo.IsNoQueryResultsError(err) {
		logger.Warn("failed to read model(s): %s", err.Error())

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return httpserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	// rely on the outside handling of access forbidden and HTTP 500
	return nil, err
}

// handleErrorOnWrite handles errors for write operations.
// Covers many default errors and responses like
//   - context.Canceled, context.DeadlineExceed -> HTTP 500
//   - db_repo.RecordNotFoundError | db_repo.NoQueryResultsError -> HTTP 404
//   - ErrModelNotChanged -> HTTP 304
//   - db.IsDuplicateEntryError -> HTTP 409
//   - validation.Error -> HTTP 400
func handleErrorOnWrite(ctx context.Context, logger log.Logger, err error) (*httpserver.Response, error) {
	logger = logger.WithContext(ctx)

	if exec.IsRequestCanceled(err) {
		logger.Error("failed to update model(s): %w", err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	if db_repo.IsRecordNotFoundError(err) || db_repo.IsNoQueryResultsError(err) {
		logger.Warn("failed to fetch model(s): %s", err.Error())

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if errors.Is(err, ErrModelNotChanged) {
		logger.Info("model(s) unchanged, rejecting update")

		return httpserver.NewStatusResponse(http.StatusNotModified), nil
	}

	if db.IsDuplicateEntryError(err) {
		return httpserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return httpserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	// rely on the outside handling of access forbidden and HTTP 500
	return nil, err
}
