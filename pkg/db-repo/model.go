package db_repo

import (
	"github.com/applike/gosoline/pkg/mdl"
	"time"
)

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

type TimeStampable interface {
	SetUpdatedAt(updatedAt *time.Time)
	SetCreatedAt(createdAt *time.Time)
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

func EmptyTimestamps() Timestamps {
	return Timestamps{
		UpdatedAt: &time.Time{},
		CreatedAt: &time.Time{},
	}
}
