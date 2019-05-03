package validation_test

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/applike/gosoline/pkg/validation"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Model struct {
	Name string
}

type NameRule struct {
}

func (NameRule) IsValid(ctx context.Context, model interface{}) error {
	m := model.(*Model)

	if len(m.Name) > 3 {
		return nil
	}

	return errors.New("name not long enough")
}

func TestValidator_IsValidDefaultGroup(t *testing.T) {
	v := buildValidator()
	v.AddRule(&NameRule{})

	m1 := &Model{
		Name: "foobar",
	}

	err := v.IsValid(context.Background(), m1)
	assert.NoError(t, err, "model should be valid")

	m2 := &Model{
		Name: "foo",
	}

	err = v.IsValid(context.Background(), m2)
	assert.Error(t, err, "model should be invalid")
}

func TestValidator_IsValidGroups(t *testing.T) {
	v := buildValidator()
	v.AddRule(&NameRule{}, "bla")

	m2 := &Model{
		Name: "foo",
	}

	err := v.IsValid(context.Background(), m2)
	assert.NoError(t, err, "model should be invalid")

	err = v.IsValid(context.Background(), m2, "bla")
	assert.Error(t, err, "model should be invalid")
}

func buildValidator() validation.Validator {
	tracer := tracing.NewNoopTracer()
	v := validation.NewValidatorWithInterfaces(tracer)

	return v
}
