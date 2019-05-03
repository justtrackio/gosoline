package db_repo

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
)

type notifyingRepository struct {
	Repository

	logger    mon.Logger
	notifiers NotificationMap
}

func NewNotifyingRepository(logger mon.Logger, base Repository) *notifyingRepository {
	return &notifyingRepository{
		Repository: base,
		logger:     logger,
		notifiers:  make(NotificationMap),
	}
}

func (r *notifyingRepository) AddNotifierAll(c Notifier) {
	for _, t := range NotificationTypes {
		r.AddNotifier(t, c)
	}
}

func (r *notifyingRepository) AddNotifier(t string, c Notifier) {
	if _, ok := r.notifiers[t]; !ok {
		r.notifiers[t] = make([]Notifier, 0)
	}

	r.notifiers[t] = append(r.notifiers[t], c)
}

func (r *notifyingRepository) Create(ctx context.Context, value ModelBased) error {
	if err := r.Repository.Create(ctx, value); err != nil {
		return err
	}

	return r.doCallback(ctx, Create, value)
}

func (r *notifyingRepository) Update(ctx context.Context, value ModelBased) error {
	if err := r.Repository.Update(ctx, value); err != nil {
		return err
	}

	return r.doCallback(ctx, Update, value)
}

func (r *notifyingRepository) Delete(ctx context.Context, value ModelBased) error {
	if err := r.Repository.Delete(ctx, value); err != nil {
		return err
	}

	return r.doCallback(ctx, Delete, value)
}

func (r *notifyingRepository) doCallback(ctx context.Context, callbackType string, value ModelBased) error {
	if _, ok := r.notifiers[callbackType]; !ok {
		return nil
	}

	errors := make([]error, 0)

	for _, c := range r.notifiers[callbackType] {
		err := c.Send(ctx, callbackType, value)

		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		err := fmt.Errorf("there were %v errors during execution of the callbacks for create", len(errors))
		r.logger.WithContext(ctx).Error(err, err.Error())

		return err
	}

	return nil
}
