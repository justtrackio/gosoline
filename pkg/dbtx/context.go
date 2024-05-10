package dbtx

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
)

type TxContext struct {
	ctx context.Context
	db  *gorm.DB
}

func (t TxContext) Deadline() (deadline time.Time, ok bool) {
	return t.ctx.Deadline()
}

func (t TxContext) Done() <-chan struct{} {
	return t.ctx.Done()
}

func (t TxContext) Err() error {
	if t.db.Error != nil {
		return t.db.Error
	}

	return t.ctx.Err()
}

func (t TxContext) Value(key any) any {
	return t.ctx.Value(key)
}

func (t TxContext) Commit() error {
	return t.db.Commit().Error
}
