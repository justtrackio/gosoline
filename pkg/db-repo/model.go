package db_repo

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

const ColumnUpdatedAt = "updated_at"

//go:generate mockery --name ModelBased
type ModelBased interface {
	mdl.Identifiable
	TimeStampable
}

type Model struct {
	Id *uint `gorm:"primary_key;AUTO_INCREMENT"`
	Timestamps
}

func (m *Model) GetId() *uint {
	return m.Id
}

//go:generate mockery --name TimeStampable
type TimeStampable interface {
	SetUpdatedAt(updatedAt *time.Time)
	SetCreatedAt(createdAt *time.Time)
}

type TimestampAware interface {
	GetCreatedAt() *time.Time
	GetUpdatedAt() *time.Time
}

type Timestamps struct {
	UpdatedAt *time.Time
	CreatedAt *time.Time
}

func (m *Timestamps) SetUpdatedAt(updatedAt *time.Time) {
	m.UpdatedAt = updatedAt
}

func (m *Timestamps) SetCreatedAt(createdAt *time.Time) {
	m.CreatedAt = createdAt
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
