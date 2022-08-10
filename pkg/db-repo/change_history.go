package db_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name ChangeHistoryModelBased
type ChangeHistoryModelBased interface {
	GetHistoryAction() string
	GetHistoryActionAt() time.Time
	GetHistoryRevision() int
}

type ChangeHistoryModel struct {
	ChangeHistoryAction   string    `gorm:"type:VARCHAR(8) NOT NULL DEFAULT 'insert'"`
	ChangeHistoryActionAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	ChangeHistoryRevision int       `gorm:"type:INT AUTO_INCREMENT;primaryKey"`
}

func (c *ChangeHistoryModel) GetHistoryAction() string {
	return c.ChangeHistoryAction
}

func (c *ChangeHistoryModel) GetHistoryActionAt() time.Time {
	return c.ChangeHistoryActionAt
}

func (c *ChangeHistoryModel) GetHistoryRevision() int {
	return c.ChangeHistoryRevision
}

func MigrateChangeHistory(ctx context.Context, config cfg.Config, logger log.Logger, models ...ModelBased) error {
	var err error
	var manager *ChangeHistoryManager

	if manager, err = ProvideChangeHistoryManager(ctx, config, logger); err != nil {
		return fmt.Errorf("can not access the change history manager: %w", err)
	}

	manager.addModels(models...)

	return nil
}
