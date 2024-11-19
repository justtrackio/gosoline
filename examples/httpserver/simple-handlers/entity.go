package main

import (
	"context"
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

func (e *MyEntity) SetUpdatedAt(updatedAt *time.Time) {
	e.UpdatedAt = updatedAt
}

func (e *MyEntity) SetCreatedAt(createdAt *time.Time) {
	e.CreatedAt = createdAt
}

type MyEntityRepository struct{}

func (*MyEntityRepository) Create(ctx context.Context, value db_repo.ModelBased) error {
	return nil
}

func (*MyEntityRepository) Read(ctx context.Context, id *uint, out db_repo.ModelBased) error {
	return nil
}

func (*MyEntityRepository) Update(ctx context.Context, value db_repo.ModelBased) error {
	return nil
}

func (*MyEntityRepository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	return nil
}

func (*MyEntityRepository) Query(ctx context.Context, qb *db_repo.QueryBuilder, result interface{}) error {
	r := result.(*[]*MyEntity)

	*r = append(*r, &MyEntity{
		Id:    1,
		Prop1: "text",
	})
	*r = append(*r, &MyEntity{
		Id:    2,
		Prop1: "text",
	})
	result = r

	return nil
}

func (*MyEntityRepository) Count(ctx context.Context, qb *db_repo.QueryBuilder, model db_repo.ModelBased) (int, error) {
	return 2, nil
}

func (*MyEntityRepository) GetMetadata() db_repo.Metadata {
	return db_repo.Metadata{
		TableName:  "entity",
		PrimaryKey: "id",
	}
}
