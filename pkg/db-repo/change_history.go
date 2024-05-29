package db_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

const (
	changeHistoryAuthorField  = "change_history_author_id"
	sqlSetChangeHistoryAuthor = "SET @" + changeHistoryAuthorField + " := ?"
)

type ChangeAuthorAware interface {
	SetChangeAuthor(authorId uint)
}

type ChangeHistoryEmbeddable struct {
	ChangeHistoryAuthorId *uint `gorm:"column:change_history_author_id"`
}

func (c *ChangeHistoryEmbeddable) SetChangeAuthor(authorId uint) {
	c.ChangeHistoryAuthorId = mdl.Box(authorId)
}

func (c *ChangeHistoryEmbeddable) BeforeCreate(tx Remote) error {
	return tx.Exec(sqlSetChangeHistoryAuthor, c.ChangeHistoryAuthorId).Error
}

func (c *ChangeHistoryEmbeddable) BeforeUpdate(tx Remote) error {
	return tx.Exec(sqlSetChangeHistoryAuthor, c.ChangeHistoryAuthorId).Error
}

func (c *ChangeHistoryEmbeddable) BeforeDelete(tx Remote) error {
	return tx.Exec(sqlSetChangeHistoryAuthor, c.ChangeHistoryAuthorId).Error
}

type ChangeHistoryModelBased interface {
	GetHistoryAction() string
	GetHistoryActionAt() time.Time
	GetHistoryRevision() int
	GetHistoryAuthorId() int
}

type ChangeHistoryModel struct {
	ChangeHistoryAction   string    `sql:"type: VARCHAR(8) NOT NULL DEFAULT 'insert'"`
	ChangeHistoryActionAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	ChangeHistoryRevision int       `gorm:"primary_key"`
	ChangeHistoryAuthorId int       `gorm:"column:change_history_author_id"`
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

func (c *ChangeHistoryModel) GetHistoryAuthorId() int {
	return c.ChangeHistoryAuthorId
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
