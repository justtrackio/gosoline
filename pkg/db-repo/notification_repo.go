package db_repo

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type notifyingRepository[K mdl.PossibleIdentifier, M ModelBased[K]] struct {
	Repository[K, M]

	logger    log.Logger
	notifiers NotificationMap[K]
}

func NewNotifyingRepository[K mdl.PossibleIdentifier, M ModelBased[K]](logger log.Logger, base Repository[K, M]) Repository[K, M] {
	return &notifyingRepository[K, M]{
		Repository: base,
		logger:     logger,
		notifiers:  make(NotificationMap[K]),
	}
}

func (r *notifyingRepository[K, M]) AddNotifierAll(c Notifier[K]) {
	for _, t := range NotificationTypes {
		r.AddNotifier(t, c)
	}
}

func (r *notifyingRepository[K, M]) AddNotifier(t string, c Notifier[K]) {
	if _, ok := r.notifiers[t]; !ok {
		r.notifiers[t] = make([]Notifier[K], 0)
	}

	r.notifiers[t] = append(r.notifiers[t], c)
}

func (r *notifyingRepository[K, M]) Create(ctx context.Context, value M) error {
	if err := r.Repository.Create(ctx, value); err != nil {
		return err
	}

	return r.doCallback(ctx, Create, value)
}

func (r *notifyingRepository[K, M]) Update(ctx context.Context, value M) error {
	if err := r.Repository.Update(ctx, value); err != nil {
		return err
	}

	return r.doCallback(ctx, Update, value)
}

func (r *notifyingRepository[K, M]) Delete(ctx context.Context, value M) error {
	if err := r.Repository.Delete(ctx, value); err != nil {
		return err
	}

	return r.doCallback(ctx, Delete, value)
}

func (r *notifyingRepository[K, M]) doCallback(ctx context.Context, callbackType string, value M) error {
	if _, ok := r.notifiers[callbackType]; !ok {
		return nil
	}

	var errors error

	for _, c := range r.notifiers[callbackType] {
		err := c.Send(ctx, callbackType, value)
		if err != nil {
			errors = multierror.Append(errors, err)
			r.logger.Warn(ctx, "%T notifier errored out with: %v", c, err)
		}
	}

	if errors != nil {
		return fmt.Errorf("there were errors during execution of the callbacks for %s: %w", callbackType, errors)
	}

	return nil
}
