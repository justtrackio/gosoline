package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/db-repo"
)

type MyEntity struct {
	Id        uint       `json:"id"`
	Prop1     string     `json:"prop1"`
	Prop2     string     `json:"prop2"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

func (e *MyEntity) GetId() *uint {
	return &e.Id
}

func (e *MyEntity) GetUpdatedAt() *time.Time {
	return e.UpdatedAt
}

func (e *MyEntity) GetCreatedAt() *time.Time {
	return e.CreatedAt
}

func (e *MyEntity) SetUpdatedAt(updatedAt *time.Time) {
	e.UpdatedAt = updatedAt
}

func (e *MyEntity) SetCreatedAt(createdAt *time.Time) {
	e.CreatedAt = createdAt
}

type MyEntityRepository struct{}

func (*MyEntityRepository) Create(ctx context.Context, value *MyEntity) error {
	return nil
}

func (*MyEntityRepository) Read(ctx context.Context, id uint) (*MyEntity, error) {
	if id == 1 {
		return &MyEntity{
			Id:    1,
			Prop1: "text",
		}, nil
	}

	if id == 2 {
		return &MyEntity{
			Id:    2,
			Prop2: "text",
		}, nil
	}

	return nil, db_repo.NewRecordNotFoundError(fmt.Sprintf("%d", id), "myEntity", fmt.Errorf("not found"))
}

func (*MyEntityRepository) Update(ctx context.Context, value *MyEntity) error {
	return nil
}

func (*MyEntityRepository) Delete(ctx context.Context, value *MyEntity) error {
	return nil
}

func (*MyEntityRepository) Query(ctx context.Context, qb *db_repo.QueryBuilder) ([]*MyEntity, error) {
	return []*MyEntity{
		{
			Id:    1,
			Prop1: "text",
		},
		{
			Id:    2,
			Prop2: "text",
		},
	}, nil
}

func (*MyEntityRepository) Count(ctx context.Context, qb *db_repo.QueryBuilder) (int, error) {
	return 2, nil
}

func (r *MyEntityRepository) GetModelId() string {
	return "project.family.name.MyEntity"
}

func (r *MyEntityRepository) GetModelName() string {
	return "MyEntity"
}

func (*MyEntityRepository) GetMetadata() db_repo.Metadata {
	return db_repo.Metadata{
		TableName:  "entity",
		PrimaryKey: "id",
	}
}
