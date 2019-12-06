package condition_test

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/condition"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Model struct {
	db_repo.Model
	MayCreate bool
	MayUpdate bool
	MayDelete bool
}

type SettingsRule struct {
}

func (SettingsRule) IsValid(ctx context.Context, model db_repo.ModelBased, operation string) error {
	m := model.(*Model)

	switch operation {
	case db_repo.Create:
		if !m.MayCreate {
			return errors.New("may not create")
		}
	case db_repo.Update:
		if !m.MayUpdate {
			return errors.New("may not update")
		}
	case db_repo.Delete:
		if !m.MayDelete {
			return errors.New("may not delete")
		}
	}

	return nil
}

func TestConditions_IsValidAllOperations(t *testing.T) {
	v := buildConditions()
	v.AddCondition(&SettingsRule{})

	m1 := &Model{
		MayCreate: true,
		MayUpdate: true,
		MayDelete: true,
	}

	err := v.IsValid(context.Background(), m1, db_repo.Create)
	assert.NoError(t, err, "operation should be valid")
	err = v.IsValid(context.Background(), m1, db_repo.Update)
	assert.NoError(t, err, "operation should be valid")
	err = v.IsValid(context.Background(), m1, db_repo.Delete)
	assert.NoError(t, err, "operation should be valid")

	m2 := &Model{
		MayCreate: false,
		MayUpdate: false,
		MayDelete: false,
	}

	err = v.IsValid(context.Background(), m2, db_repo.Create)
	assert.Error(t, err, "operation should be invalid")
	err = v.IsValid(context.Background(), m2, db_repo.Update)
	assert.Error(t, err, "operation should be invalid")
	err = v.IsValid(context.Background(), m2, db_repo.Delete)
	assert.Error(t, err, "operation should be invalid")
}

func TestConditions_IsValidSomeOperations(t *testing.T) {
	v := buildConditions()
	v.AddCondition(&SettingsRule{}, db_repo.Create, db_repo.Delete)

	m1 := &Model{
		MayCreate: true,
		MayUpdate: false,
		MayDelete: true,
	}

	err := v.IsValid(context.Background(), m1, db_repo.Create)
	assert.NoError(t, err, "operation should be valid")
	err = v.IsValid(context.Background(), m1, db_repo.Update)
	assert.NoError(t, err, "operation should be valid")
	err = v.IsValid(context.Background(), m1, db_repo.Delete)
	assert.NoError(t, err, "operation should be valid")

	m2 := &Model{
		MayCreate: false,
		MayUpdate: false,
		MayDelete: false,
	}

	err = v.IsValid(context.Background(), m2, db_repo.Create)
	assert.Error(t, err, "operation should be invalid")
	err = v.IsValid(context.Background(), m2, db_repo.Delete)
	assert.Error(t, err, "operation should be invalid")
}

func buildConditions() condition.Conditions {
	tracer := tracing.NewNoopTracer()
	v := condition.NewConditionsWithInterfaces(tracer)

	return v
}
