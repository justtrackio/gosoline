package db_repo

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

const ColumnUpdatedAt = "updated_at"

//go:generate mockery --name ModelBased
type ModelBased interface {
	mdl.Identifiable
}

type Model struct {
	Id *uint `gorm:"primaryKey;type:int unsigned AUTO_INCREMENT"`
	Timestamps
}

func (m *Model) GetId() *uint {
	return m.Id
}

//go:generate mockery --name TimestampAware
type TimestampAware interface {
	GetCreatedAt() *time.Time
	GetUpdatedAt() *time.Time
}

type Timestamps struct {
	UpdatedAt *time.Time `gorm:"autoCreateTime"`
	CreatedAt *time.Time `gorm:"autoUpdateTime"`
}

func (m *Timestamps) GetUpdatedAt() *time.Time {
	return m.UpdatedAt
}

func (m *Timestamps) GetCreatedAt() *time.Time {
	return m.CreatedAt
}

func EmptyTimestamps() Timestamps {
	return Timestamps{
		UpdatedAt: &time.Time{},
		CreatedAt: &time.Time{},
	}
}
