package db_repo

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
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

func MigrateChangeHistory(config cfg.Config, logger mon.Logger, models ...ModelBased) {
	for _, model := range models {
		manager := newChangeHistoryManager(config, logger, model)
		err := manager.runMigration()
		if err != nil {
			panic(err)
		}
	}
}
