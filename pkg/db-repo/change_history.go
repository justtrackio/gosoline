package db_repo

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"time"
)

type ChangeHistoryModelBased interface {
	GetHistoryAction() string
	GetHistoryActionAt() time.Time
	GetHistoryRevision() int
}

type ChangeHistoryModel struct {
	ChangeHistoryAction   string    `sql:"type: VARCHAR(8) NOT NULL DEFAULT 'insert'"`
	ChangeHistoryActionAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	ChangeHistoryRevision int       `gorm:"primary_key"`
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

func MigrateChangeHistory(config cfg.Config, logger log.Logger, models ...ModelBased) error {
	for _, model := range models {
		manager, err := newChangeHistoryManager(config, logger, model)
		if err != nil {
			return fmt.Errorf("can not create change history manager: %w", err)
		}

		if err = manager.runMigration(); err != nil {
			return fmt.Errorf("can not run migrations: %w", err)
		}
	}

	return nil
}
